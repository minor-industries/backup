package main

import (
	"fmt"
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
		parsedURL.User = url.UserPassword(username, "XXXX")
	}

	return prefix + parsedURL.String(), nil
}

func main() {
	inputURL := "rest:https://user:pass@host.vpn:8074/path"
	maskedURL, err := maskPassword(inputURL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(maskedURL)
}
