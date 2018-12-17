package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/pkg/errors"
)

// LogFormat is the default parsing format
// Apache-Style: "%h %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\""

// AccessLog represents the info embedded in a log line.
type AccessLog struct {
	// %h
	RemoteHostname string
	// %l
	RemoteLogname string
	// %u
	RemoteUser string
	// %t
	Time string
	// "%r"
	Request Request
	// %>s (final status)
	StatusCode int
	// %O (includes headers)
	BytesSent int
	// "%{Referer}i"
	Referer string
	// "%{User-Agent}i"
	UserAgent string
}

// AccessLogList adds methods to a slice of AccessLog
type AccessLogList struct {
	Logs []AccessLog
}

// Request is the log's parsed request line
type Request struct {
	Method  string
	URI     string
	Version string
}

/*
LogFormat "%h %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\""
Example: 73.92.251.192 - - [16/Dec/2018:06:25:09 +0000] "GET /learn/at-what-age-should-i-start-making-401k-withdrawals/ HTTP/1.1" 200 14687 "https://www.google.com/" "Mozilla/5.0 (iPhone; CPU iPhone OS 12_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1"
*/

// Digest consumes the raw input and sets its fields from parsing it.
func (aLog *AccessLog) Digest(input []byte) error {
	state := 0

	for ; len(input) > 0; state++ {
		switch state {
		case 0: // %h
			i, b := extractUntil(input, ' ')
			aLog.RemoteHostname = string(transformNilLogItem(b))
			input = input[i:]

		case 1: // %l
			i, b := extractUntil(input, ' ')
			aLog.RemoteLogname = string(transformNilLogItem(b))
			input = input[i:]

		case 2: // %u
			i, b := extractUntil(input, ' ')
			aLog.RemoteUser = string(transformNilLogItem(b))
			input = input[i:]

		case 3: // %t
			i, b, err := extractWrappedUntil(input, '[', ']', ' ')
			if err != nil {
				return errors.Wrap(err, "failed to parse time")
			}
			aLog.Time = string(transformNilLogItem(b))
			input = input[i:]

		case 4: // "%r"
			i, b, err := extractWrappedUntil(input, '"', '"', ' ')
			if err != nil {
				return errors.Wrap(err, "failed to parse request")
			}
			input = input[i:]

			aLog.Request = Request{}
			if err := aLog.Request.Digest(b); err != nil {
				return err
			}

		case 5: // %>s
			i, b := extractUntil(input, ' ')
			status, err := strconv.Atoi(string(b))
			if err != nil {
				return errors.Wrapf(err, "failed to convert status %q to integer", string(b))
			}

			aLog.StatusCode = status
			input = input[i:]

		case 6: // %O
			i, b := extractUntil(input, ' ')
			bytesSent, err := strconv.Atoi(string(b))
			if err != nil {
				return errors.Wrapf(err, "failed to convert bytes sent %q to integer", string(b))
			}
			aLog.BytesSent = bytesSent
			input = input[i:]

		case 7: // "%{Referer}i"
			i, b, err := extractWrappedUntil(input, '"', '"', ' ')
			if err != nil {
				return errors.Wrapf(err, "failed to parse Referer from: %q", string(input))
			}
			aLog.Referer = string(b)
			input = input[i:]

		case 8: // "%{User-Agent}i"
			// Take the rest
			aLog.UserAgent = string(bytes.Trim(input, `"`))
			input = input[len(input):]
		}
	}

	return nil
}

// Digest consumes the input and sets parsed values on the Request.
func (r *Request) Digest(input []byte) error {
	if bytes.Equal(input, []byte{'-'}) {
		return nil
	}

	buf := bytes.NewBuffer(input)

	_, err := fmt.Fscanf(buf, "%s %s %s", &r.Method, &r.URI, &r.Version)
	if err != nil {
		return errors.Wrapf(err, "failed parsing original request: %q", string(input))
	}

	return nil
}

// extracts a slice of bytes up to a delimiter,
// also returning the next index (following the delimiter)
func extractUntil(input []byte, delimiter byte) (int, []byte) {
	i := bytes.IndexByte(input, delimiter)
	if i == -1 {
		return len(input), input[:]
	}

	return i + 1, input[0:i]
}

// extracts a slice of bytes enclosed in left/right tokens up to a delimiter,
// also returning the next index (following the delimiter)
func extractWrappedUntil(input []byte, left, right, delimiter byte) (int, []byte, error) {
	i := bytes.IndexByte(input, left)
	if i != 0 {
		return i, nil, errors.Errorf(
			"invalid start character %q: expecting starting character %q",
			string(input[0]), string(left),
		)
	}

	if left == right {
		// Same token for left & right, so look after the 1st occurrence
		i = bytes.IndexByte(input[1:], right)
		// Adjust the index value for the re-slice
		i++
	} else {
		i = bytes.IndexByte(input, right)
	}

	if i == -1 {
		return i, nil, errors.Errorf(
			"could not find end of sequence: expecting terminating character %q",
			string(right),
		)
	}

	// Input ends at right token, return everything inside tokens
	if len(input) == i+1 {
		return i + 1, input[1:i], nil
	}

	if input[i+1] != delimiter {
		return i, nil, errors.Errorf(
			"end of sequence not followed by delimiter %q",
			string(delimiter),
		)
	}

	return i + 2, input[1:i], nil
}

func transformNilLogItem(b []byte) []byte {
	if bytes.Equal(b, []byte{'-'}) {
		return nil
	}

	return b
}

// KVPair is a key value pair
type KVPair struct {
	Key   string
	Value int
}

func filterSort(countMap map[string]int, min, max int) []KVPair {
	pairs := make([]KVPair, 0, len(countMap))

	for k, v := range countMap {
		if v < min || (max > 0 && v > max) {
			continue
		}
		pairs = append(pairs, KVPair{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	return pairs
}
