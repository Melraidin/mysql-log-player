package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

func main() {
	runtime.GOMAXPROCS(1)

	reader := bufio.NewReader(os.Stdin)
	filename := "19700101-00"; ext := ".gz"
	var f *os.File
	var bw *bufio.Writer
	var writer *gzip.Writer

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			close(writer, bw, f)

			if err == io.EOF {
				os.Exit(0)
			}
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if strings.HasPrefix(line, "# Time: ") {
			mmddyy_hh_mm := line[8:]
			dateHour := fmt.Sprintf("20%s%s%s-%s", mmddyy_hh_mm[4:6], mmddyy_hh_mm[:2], mmddyy_hh_mm[2:4], mmddyy_hh_mm[7:9])
			if dateHour > filename {
				close(writer, bw, f)

				filename = dateHour

				err = open(filename + ext, &f, &bw, &writer)
				exitOnError(err, writer, bw, f)
			}
		}
		writer.Write([]byte(line))
	}
}

func open(filename string, f **os.File, bw **bufio.Writer, writer **gzip.Writer) (err error) {
	if *f, err = os.Create(filename); err != nil {
		return
	}
	*bw = bufio.NewWriter(*f)
	*writer, err = gzip.NewWriterLevel(*bw, gzip.BestCompression)
	return
}

func close(writer *gzip.Writer, bw *bufio.Writer, f *os.File) {
	if writer != nil {
		writer.Close()
	}
	if bw != nil {
		bw.Flush()
	}
	if f != nil {
		f.Close()
	}
}

func exitOnError(err error, writer *gzip.Writer, bw *bufio.Writer, f *os.File) {
	if err != nil {
		close(writer, bw, f)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
