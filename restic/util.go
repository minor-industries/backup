package restic

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
		if _, hasPassword := parsedURL.User.Password(); hasPassword {
			parsedURL.User = url.UserPassword(username, "XXXX")
		} else {
			parsedURL.User = url.User(username)
		}
	}

	return prefix + parsedURL.String(), nil
}

func QuantizeFilter(callback func(msg any) error) func(msg any) error {
	lastQuantum := -1.0

	return func(msg any) error {
		switch msg := msg.(type) {
		case ResticStatus:
			currentQuantum := float64(int(msg.PercentDone*10)) / 10.0
			if currentQuantum > lastQuantum {
				lastQuantum = currentQuantum
				return callback(msg)
			}
			return nil
		case StartBackup, ResticSummary:
			lastQuantum = -1.0
			return callback(msg)
		default:
			return callback(msg)
		}
	}
}

func LogMessages(callback func(msg string) error) func(msg any) error {
	return QuantizeFilter(func(msg any) error {
		switch msg := msg.(type) {
		case StartBackup:
			if msg.KeychainProfile != "" {
				return callback(fmt.Sprintf("loading keychain profile: %s", msg.KeychainProfile))
			}
			if msg.Repository != "" {
				return callback(fmt.Sprintf("starting backup: %s", msg.Repository))
			}
		case ResticStatus:
			return callback(fmt.Sprintf("progress: %.1f%%", msg.PercentDone*100))
		case ResticSummary:
			return callback("backup done")
		default:
			return callback("unknown message type")
		}
		return nil
	})
}
