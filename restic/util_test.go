package restic

import (
	"testing"
)

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "rest:https://user:pass@host.vpn:8074/path",
			expected: "rest:https://user:XXXX@host.vpn:8074/path",
		},
		{
			input:    "rest:https://user:p@host.vpn:8074/path",
			expected: "rest:https://user:XXXX@host.vpn:8074/path",
		},
		{
			input:    "rest:https://user:password123@host.vpn:8074/path",
			expected: "rest:https://user:XXXX@host.vpn:8074/path",
		},
		{
			input:    "rest:https://user@host.vpn:8074/path",
			expected: "rest:https://user@host.vpn:8074/path",
		},
		{
			input:    "s3:s3.us-east-1.amazonaws.com/bucket_name",
			expected: "s3:s3.us-east-1.amazonaws.com/bucket_name",
		},
		{
			input:    "path/to/backup",
			expected: "path/to/backup",
		},
		{
			input:    "/abs/path/to/backup",
			expected: "/abs/path/to/backup",
		},
	}

	for _, test := range tests {
		result, err := maskPassword(test.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != test.expected {
			t.Errorf("for input %q, expected %q but got %q", test.input, test.expected, result)
		}
	}
}
