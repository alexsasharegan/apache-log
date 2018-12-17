package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/pkg/errors"
)

var usage = `
Apache Log Utils

Usage:  apache-log ...[input]

`

func main() {
	stderr := log.New(os.Stderr, "", 0)
	flag.Parse()
	args := flag.Args()
	// No input files received
	if len(args) == 0 {
		stderr.Print(usage)
		os.Exit(1)
	}

	logx := make(chan []AccessLog)
	notFound := make(map[string]int)
	var wg sync.WaitGroup

	wg.Add(len(args))
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
		close(logx)
	}()

	for entries := range logx {
		for _, aLog := range entries {
			if aLog.StatusCode == 404 {
				notFound[aLog.Request.URI]++
			}
		}
	}

	var buf bytes.Buffer
	for _, pair := range rankByCount(notFound, 10) {
		buf.WriteString(fmt.Sprintf("%d: %s\n", pair.Value, pair.Key))
	}

	buf.WriteTo(os.Stdout)
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

// KVPair is a key value pair
type KVPair struct {
	Key   string
	Value int
}

// KVPairList implements the sort interface
type KVPairList []KVPair

func (p KVPairList) Len() int           { return len(p) }
func (p KVPairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p KVPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func rankByCount(countMap map[string]int, threshold int) []KVPair {
	pairs := make([]KVPair, 0, len(countMap))

	for k, v := range countMap {
		if v < threshold {
			continue
		}
		pairs = append(pairs, KVPair{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	return pairs
}
