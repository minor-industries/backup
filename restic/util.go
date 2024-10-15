package restic

import (
	"net/url"
	"regexp"
)

func maskPassword(input string) (string, error) {
	re := regexp.MustCompile(`^([a-zA-Z]+:)?(.*)`)
	matches := re.FindStringSubmatch(input)
	prefix := matches[1]
	strippedInput := matches[2]

	parsedURL, err := url.Parse(strippedInput)
	if err != nil {
		return "", err
	}

	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		if _, hasPassword := parsedURL.User.Password(); hasPassword {
			parsedURL.User = url.UserPassword(username, "XXXX")
		} else {
			parsedURL.User = url.User(username)
		}
	}

	return prefix + parsedURL.String(), nil
}
