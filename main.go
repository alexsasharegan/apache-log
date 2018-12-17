package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/alexsasharegan/apache-log/timing"
	"github.com/pkg/errors"
)

var usage = `Apache Log Utils

    apache-log [flags] ...[input]

`

// Performance tracks timing for application tasks.
type Performance struct {
	Execution timing.Timing
	Parsing   timing.Timing
	Sorting   timing.Timing
}

// ProgramFlags are the command line flags.
type ProgramFlags struct {
	verbose *bool
	status  *int
	max     *int
	min     *int
	exclude *string
}

var program = ProgramFlags{
	verbose: flag.Bool("v", false, "verbose log output"),
	status:  flag.Int("status", 200, "filter by status code"),
	max:     flag.Int("max", 0, "filters entries exceeding max occurrence"),
	min:     flag.Int("min", 0, "filters entries not meeting min occurrence"),
	exclude: flag.String("x", "", "exclude URLs starting with string (comma separated for multiple)"),
}

func main() {
	var perf Performance
	perf.Execution.Start()
	stderr := log.New(os.Stderr, "", 0)

	flag.Parse()
	args := flag.Args()
	// No input files received
	if len(args) == 0 {
		stderr.Print(usage)
		flag.Usage()
		os.Exit(1)
	}

	logx := make(chan []AccessLog)
	byStatus := make(map[string]int)

	var wg sync.WaitGroup
	wg.Add(len(args))

	perf.Parsing.Start()
	for _, filename := range args {
		go func(name string) {
			defer wg.Done()
			entries, err := parseFile(name)
			if err != nil {
				stderr.Fatalln(err)
			}

			logx <- entries
		}(filename)
	}

	go func() {
		wg.Wait()
		perf.Parsing.Stop()
		close(logx)
	}()

	perf.Sorting.Start()
	statusCode := *program.status
	checkExclude := len(*program.exclude) > 0
	exclude := strings.Split(*program.exclude, ",")

	for entries := range logx {
	ENTRIES_LOOP:
		for _, aLog := range entries {
			if aLog.StatusCode != statusCode {
				continue
			}

			if checkExclude {
				for _, x := range exclude {
					if strings.HasPrefix(aLog.Request.URI, x) {
						continue ENTRIES_LOOP
					}
				}
			}

			byStatus[aLog.Request.URI]++
		}
	}
	perf.Sorting.Stop()

	var buf bytes.Buffer
	for _, pair := range filterSort(byStatus, *program.min, *program.max) {
		buf.WriteString(fmt.Sprintf("%d: %s\n", pair.Value, pair.Key))
	}

	buf.WriteTo(os.Stdout)

	perf.Execution.Stop()
	if *program.verbose {
		stderr.Println("Parsing:", perf.Parsing.ElapsedString())
		stderr.Println("Sorting:", perf.Sorting.ElapsedString())
		stderr.Println("Total  :", perf.Execution.ElapsedString())
	}
}

func parseFile(name string) ([]AccessLog, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open file %q", name)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "could not stat file %q", name)
	}

	// Log files can be quite large,
	// so alloc capacity by average length of log line (guestimate)
	entries := make([]AccessLog, 0, finfo.Size()/256)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		aLog := AccessLog{}
		err := aLog.Digest(scanner.Bytes())
		if err != nil {
			return nil, err
		}

		entries = append(entries, aLog)
	}

	return entries, scanner.Err()
}
