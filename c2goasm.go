package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "//+build !noasm !appengine")
	fmt.Fprintln(w, "// AUTO-GENERATED BY C2GOASM -- DO NOT EDIT")
	fmt.Fprintln(w, "")
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func process(assembly []string) ([]string, error) {

	// TODO
	// strip header
	// add golang header
	// consistent use of rbp & rsp
	// test for absence of CALLs

	// Get one segment per function
	segments := SegmentSource(assembly)
	tables := SegmentConsts(assembly)

	var result []string

	// Iterate over all subroutines
	for isegment, s := range segments {

		argsOnStack := ArgumentsOnStack(assembly[s.Start:s.End])
		fmt.Println("ARGUMENTS ON STACK", argsOnStack)

		// Check for constants table
		var table Table
		if table = GetCorrespondingTable(assembly[s.Start:s.End], tables); table.IsPresent() {

			// Output constants table
			result = append(result, strings.Split(table.Data, "\n")...)
			result = append(result, "")	// append empty line
		}

		// Define subroutine

		result = append(result, WriteGoasmPrologue(s, 6, table)...)

		// Write body of code
		assembly, err := assemblify(assembly[s.Start:s.End], table, s.stack)
		if err != nil {
			panic(fmt.Sprintf("assemblify error: %v", err))
		}
		result = append(result, assembly...)

		// Return from subroutine
		result = append(result, s.stack.WriteGoasmEpilogue()...)

		if isegment < len(segments)-1 {
			// Empty lines before next subroutine
			result = append(result, "\n", "\n")
		}
	}

	return result, nil
}

func main() {

	if len(os.Args) < 3 {
		fmt.Printf("error: no input files specified\n\n")
		fmt.Println("usage: c2goasm /path/to/c-project/build/SomeGreatCode.cpp.s SomeGreatCode_amd64.s")
		return
	}
	fmt.Println("Processing", os.Args[1])
	lines, err := readLines(os.Args[1])
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	result, err := process(lines)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}

	err = writeLines(result, os.Args[2])
	if err != nil {
		log.Fatalf("writeLines: %s", err)
	}
}
