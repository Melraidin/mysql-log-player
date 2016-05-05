package query

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
	"time"
	"fmt"
)

var (
	clientRegex = regexp.MustCompile("^# User@Host: [^ ]+ @ ([^ ]+) \\[]\n$")
)

type Reader struct {
	source io.ReadCloser
	reader *bufio.Reader
	next   string
	eof    bool
	buffer bytes.Buffer
}

var (
	zeroTime = time.Time{}
)

func NewReader(source io.ReadCloser) (*Reader, error) {
	return &Reader{
			source: source,
			reader: bufio.NewReader(source),
		},
		nil
}

func (r *Reader) Read() (*Query, error) {
	if r.eof {
		return nil, io.EOF
	}

	query := &Query{}
	for {
		var err error
		var line string
		if r.next != "" {
			line = r.next
			r.next = ""
		} else {
			line, err = r.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					r.eof = true
					query.SQL = r.getBuffer()
					return query, nil
				}
				return nil, err
			}
		}

		if strings.HasPrefix(line, "# Time: ") {
			if query.Time == zeroTime {
				query.Time, err = time.ParseInLocation("010206 15:04:05", strings.TrimSpace(line[8:]), time.UTC)
				if err != nil {
					fmt.Printf("Error parsing time '%s': %v\n", line[8:], err)
				}
			} else {
				r.next = line
				query.SQL = r.getBuffer()
				return query, nil
			}
		} else if strings.HasPrefix(line, "# User@Host: ") {
			if matches := clientRegex.FindStringSubmatch(line); len(matches) == 2 {
				query.Client = matches[1]
			}
		} else if strings.HasPrefix(line, "# Query_time: ") {
		} else {
			r.buffer.WriteString(line)
		}
	}
}

func (r *Reader) getBuffer() string {
	s := strings.TrimRight(r.buffer.String(), "\n")
	r.buffer.Reset()
	return s
}

func (r *Reader) Close() {
	r.source.Close()
}
