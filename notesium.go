package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Link struct {
	Filename   string
	LineNumber int
}

type Note struct {
	Filename      string
	Title         string
	IsLabel       bool
	OutgoingLinks []Link
	IncomingLinks []Link
	Ctime         time.Time
	Mtime         time.Time
}

type SortByCtime []*Note
type SortByMtime []*Note
type SortByTitle []*Note

func (n SortByCtime) Len() int           { return len(n) }
func (n SortByCtime) Less(i, j int) bool { return n[i].Ctime.After(n[j].Ctime) }
func (n SortByCtime) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func (n SortByMtime) Len() int           { return len(n) }
func (n SortByMtime) Less(i, j int) bool { return n[i].Mtime.After(n[j].Mtime) }
func (n SortByMtime) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func (n SortByTitle) Len() int           { return len(n) }
func (n SortByTitle) Less(i, j int) bool { return n[i].Title < n[j].Title }
func (n SortByTitle) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

var noteCache map[string]*Note
var linkRegex = regexp.MustCompile(`\]\(([0-9a-f]{8}\.md)\)`)

func main() {
	helpFlags := map[string]bool{"-h": true, "--help": true, "help": true}
	if len(os.Args) < 2 || helpFlags[os.Args[1]] {
		usage()
	}

	notesiumDir, err := getNotesiumDir()
	if err != nil {
		fatal("%v", err)
	}

	switch os.Args[1] {
	case "home":
		fmt.Println(notesiumDir)
	case "new":
		notesiumNew(notesiumDir)
	case "list":
		var limit, prefix, sortBy string
		for _, arg := range os.Args[2:] {
			switch {
			case arg == "--labels":
				limit = "labels"
			case arg == "--orphans":
				limit = "orphans"
			case strings.HasPrefix(arg, "--prefix="):
				prefix = strings.TrimPrefix(arg, "--prefix=")
			case strings.HasPrefix(arg, "--sort="):
				sortBy = strings.TrimPrefix(arg, "--sort=")
			}
		}
		notesiumList(notesiumDir, limit, prefix, sortBy)
	case "links":
		var filename, limit string
		filenameRequired := false
		for _, arg := range os.Args[2:] {
			switch {
			case arg == "--dangling":
				limit = "dangling"
			case arg == "--outgoing":
				limit = map[bool]string{true: "", false: "outgoing"}[limit == "incoming"]
				filenameRequired = true
			case arg == "--incoming":
				limit = map[bool]string{true: "", false: "incoming"}[limit == "outgoing"]
				filenameRequired = true
			case strings.HasSuffix(arg, ".md"):
				filename = arg
			}
		}
		if limit == "dangling" && filename != "" {
			fatal("dangling filename not supported")
		}
		if filenameRequired && filename == "" {
			fatal("filename not specified")
		}
		notesiumLinks(notesiumDir, filename, limit)
	default:
		fatal("unrecognized command: %s", os.Args[1])
	}
}

func notesiumNew(dir string) {
	epochInt := time.Now().Unix()
	epochHex := fmt.Sprintf("%x", epochInt)
	fmt.Printf("%s/%s.md\n", dir, epochHex)
}

func notesiumList(dir string, limit string, prefix string, sortBy string) {
	populateCache(dir)
	notes := getSortedNotes(sortBy)

	switch limit {
	case "labels":
		for _, note := range notes {
			if note.IsLabel {
				fmt.Printf("%s:1: %s\n", note.Filename, note.Title)
			}
		}
		return
	case "orphans":
		for _, note := range notes {
			if len(note.OutgoingLinks) == 0 && len(note.IncomingLinks) == 0 {
				fmt.Printf("%s:1: %s\n", note.Filename, note.Title)
			}
		}
		return
	}

	switch prefix {
	case "label":
		var notesWithoutLabelLinks []*Note
		var outputLines []string
		for _, note := range notes {
			labelLinked := false
			for _, link := range note.OutgoingLinks {
				if linkNote, exists := noteCache[link.Filename]; exists && linkNote.IsLabel {
					line := fmt.Sprintf("%s:1: %s %s", note.Filename, linkNote.Title, note.Title)
					if sortBy == "alpha" {
						outputLines = append(outputLines, line)
					} else {
						fmt.Println(line)
					}
					labelLinked = true
				}
			}
			if !labelLinked {
				notesWithoutLabelLinks = append(notesWithoutLabelLinks, note)
			}
		}
		if sortBy == "alpha" {
			sort.Slice(outputLines, func(i, j int) bool {
				sub_i := strings.SplitN(outputLines[i], ": ", 2)[1]
				sub_j := strings.SplitN(outputLines[j], ": ", 2)[1]
				return sub_i < sub_j
			})
			for _, line := range outputLines {
				fmt.Println(line)
			}
		}
		for _, note := range notesWithoutLabelLinks {
			fmt.Printf("%s:1: %s\n", note.Filename, note.Title)
		}
		return
	case "ctime":
		for _, note := range notes {
			fmt.Printf("%s:1: %s %s\n", note.Filename, note.Ctime.Format("2006-01-02"), note.Title)
		}
		return
	case "mtime":
		for _, note := range notes {
			fmt.Printf("%s:1: %s %s\n", note.Filename, note.Mtime.Format("2006-01-02"), note.Title)
		}
		return
	}

	for _, note := range notes {
		fmt.Printf("%s:1: %s\n", note.Filename, note.Title)
	}
}

func notesiumLinks(dir string, filename string, limit string) {
	populateCache(dir)

	if filename != "" {
		note, exists := noteCache[filename]
		if !exists {
			log.Fatalf("filename does not exist")
		}
		switch limit {
		case "outgoing":
			for _, link := range note.OutgoingLinks {
				linkNote, exists := noteCache[link.Filename]
				if exists {
					fmt.Printf("%s:1: %s\n", linkNote.Filename, linkNote.Title)
				}
			}
			return
		case "incoming":
			for _, link := range note.IncomingLinks {
				linkNote, exists := noteCache[link.Filename]
				if exists {
					fmt.Printf("%s:%d: %s\n", linkNote.Filename, link.LineNumber, linkNote.Title)
				}
			}
			return
		default:
			for _, link := range note.OutgoingLinks {
				linkNote, exists := noteCache[link.Filename]
				if exists {
					fmt.Printf("%s:1: outgoing %s\n", linkNote.Filename, linkNote.Title)
				}
			}
			for _, link := range note.IncomingLinks {
				linkNote, exists := noteCache[link.Filename]
				if exists {
					fmt.Printf("%s:%d: incoming %s\n", linkNote.Filename, link.LineNumber, linkNote.Title)
				}
			}
		}
		return
	}

	switch limit {
	case "dangling":
		for _, note := range noteCache {
			for _, link := range note.OutgoingLinks {
				_, exists := noteCache[link.Filename]
				if !exists {
					fmt.Printf("%s:%d: %s → %s\n", note.Filename, link.LineNumber, note.Title, link.Filename)
				}
			}
		}
		return
	}

	for _, note := range noteCache {
		for _, link := range note.OutgoingLinks {
			linkNote, exists := noteCache[link.Filename]
			linkTitle := link.Filename
			if exists {
				linkTitle = linkNote.Title
			}
			fmt.Printf("%s:%d: %s → %s\n", note.Filename, link.LineNumber, note.Title, linkTitle)
		}
	}
}

func populateCache(dir string) {
	noteCache = make(map[string]*Note)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("Could not read directory: %s\n", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			filename := file.Name()
			note, err := readNote(dir, filename)
			if err != nil {
				log.Fatalf("Could not read note: %s\n", err)
			}
			noteCache[filename] = note
		}
	}

	for _, note := range noteCache {
		for _, link := range note.OutgoingLinks {
			if targetNote, exists := noteCache[link.Filename]; exists {
				targetNote.IncomingLinks = append(targetNote.IncomingLinks, Link{
					LineNumber: link.LineNumber,
					Filename:   note.Filename,
				})
			}
		}
	}
}

func readNote(dir string, filename string) (*Note, error) {
	path := filepath.Join(dir, filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %s", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not get file info: %s", err)
	}
	mtime := info.ModTime()

	hexTime := strings.TrimSuffix(filename, ".md")
	unixTime, err := strconv.ParseInt(hexTime, 16, 64)
	if err != nil {
		return nil, err
	}
	ctime := time.Unix(unixTime, 0)

	var title string
	var isLabel bool
	var outgoingLinks []Link

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if title == "" {
			title = strings.TrimPrefix(line, "# ")
			isLabel = len(strings.Fields(title)) == 1
			continue
		}
		matches := linkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			outgoingLinks = append(outgoingLinks, Link{LineNumber: lineNumber, Filename: match[1]})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	note := &Note{
		Filename:      filename,
		Title:         title,
		IsLabel:       isLabel,
		OutgoingLinks: outgoingLinks,
		Ctime:         ctime,
		Mtime:         mtime,
	}

	return note, nil
}

func getSortedNotes(sortBy string) []*Note {
	notes := make([]*Note, 0, len(noteCache))
	for _, note := range noteCache {
		notes = append(notes, note)
	}

	switch sortBy {
	case "ctime":
		sort.Sort(SortByCtime(notes))
	case "mtime":
		sort.Sort(SortByMtime(notes))
	case "alpha":
		sort.Sort(SortByTitle(notes))
	}
	return notes
}

func getNotesiumDir() (string, error) {
	dir, exists := os.LookupEnv("NOTESIUM_DIR")
	if !exists {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "notes")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return "", fmt.Errorf("NOTESIUM_DIR does not exist: %s", absDir)
	}
	info, err := os.Stat(realDir)
	if err != nil {
		return "", fmt.Errorf("NOTESIUM_DIR does not exist: %s", realDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("NOTESIUM_DIR is not a directory: %s", realDir)
	}

	return realDir, nil
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Fatal: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Printf(`Usage: %s COMMAND [OPTIONS]

Commands:
  new               Print path for a new note
  home              Print path to notes directory
  list              Print list of notes
    --labels        Limit list to only label notes (ie. one word title)
    --orphans       Limit list to notes without outgoing or incoming links
    --sort=WORD     Sort list by date or alphabetically (ctime|mtime|alpha)
    --prefix=WORD   Prefix title with date or linked label (ctime|mtime|label)
  links [filename]  Print list of links
    --outgoing      Limit list to outgoing links related to filename
    --incoming      Limit list to incoming links related to filename
    --dangling      Limit list to broken links

Environment:
  NOTESIUM_DIR      Path to notes directory (default: $HOME/notes)

`, filepath.Base(os.Args[0]))
	os.Exit(1)
}
