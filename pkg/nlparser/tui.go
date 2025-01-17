package nlparser

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	generalHelpMessage = `Commands:
    (help|?): show help text
    (parse|p) <file>: parse a new file
    (show|s) <arg>: show something of the parsed netlog (use show help for more)
    (extract|e) <arg>: extract events of the parsed netlog (use show help for more)
    (quit|q): exit`
	showCommandHelpMessage = `Options:
    range: events ids range
    id <number>: print event with given id
    all: dumps all events may. this will output lots of data
    next: print the next event based on print counter
    prev: print the prev event based on print counter
    set <number>: set the print counter`
	extractCommandHelpMessage = `Options:
      dns: print all dns queries
      url: print all url requests
      red: print all redirections
      con: print all connections
      src <path>: save all sources in the given directory. You also need to have captured the netlog with the --net-log-capture-mode=Everything flag.`
	invalidCommandMessage = `invalid command`
	helpCase              = "help"
	maxFilenameLength     = 255
)

// TODO: move this counter to a struct
var pc = 0

// Help returns the program instructions
func Help() string {
	return generalHelpMessage
}

// HandleCommand parse and execute command on the given netlog
func HandleCommand(command string, netlog *NetLog) (string, error) {
	pieces := strings.Split(command, " ")
	if len(pieces) < 1 {
		return invalidCommandMessage, nil
	}
	switch pieces[0] {
	case "?":
		fallthrough
	case helpCase:
		return Help(), nil
	case "p":
		fallthrough
	case "parse":
		if len(pieces) < 2 {
			return invalidCommandMessage, nil
		}
		nl, err := ParseFile(pieces[1])
		// TODO: reconsider this
		*netlog = nl
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("File parsed. Found %d events.\n", len(netlog.Events)), nil
	case "s":
		fallthrough
	case "show":
		if len(pieces) < 2 {
			return showCommandHelpMessage, nil
		}
		return handleShow(pieces[1:], netlog)
	case "e":
		fallthrough
	case "extract":
		if len(pieces) < 2 {
			return extractCommandHelpMessage, nil
		}
		return handleExtract(pieces[1:], netlog)
	case "q":
		fallthrough
	case "quit":
		os.Exit(0)
	default:
		return invalidCommandMessage, nil
	}
	return invalidCommandMessage, nil
}

// ParseFile parse a netlog json file
func ParseFile(s string) (NetLog, error) {
	pc = 0
	info, err := os.Stat(s)
	if err != nil {
		return NetLog{}, err
	}
	if info.IsDir() {
		return NetLog{}, fmt.Errorf("%s is not a valid file", s)
	}
	return ParseNetLog(s)
}

func handleShow(args []string, netlog *NetLog) (string, error) {
	switch args[0] {
	case "range":
		return fmt.Sprintf("IDs range from 1 to %d", len(netlog.Events)), nil
	case "id":
		if len(args) < 2 {
			return invalidCommandMessage, nil
		}
		index, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return "", err
		}
		e, ok := netlog.Events[int(index)]
		if !ok {
			return fmt.Sprintf("IDs range from 0 to %d", len(netlog.Events)), nil
		}
		return fmt.Sprintf("%s", e), nil
	case "all":
		out := ""
		for k := range netlog.Events {
			out += fmt.Sprintln(netlog.Events[k])
		}
		return out, nil
	case "next":
		pc = pc + 1
		if pc > len(netlog.Events) {
			pc = 1
		}
		return fmt.Sprintln(netlog.Events[pc]), nil
	case "prev":
		pc = pc - 1
		if pc <= 0 {
			pc = len(netlog.Events)
		}
		return fmt.Sprintln(netlog.Events[pc]), nil
	case "set":
		if len(args) < 2 {
			return invalidCommandMessage, nil
		}
		index, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return "", err
		}
		pc = int(index)
		return "position updated", nil
	case helpCase:
		return showCommandHelpMessage, nil
	default:
		return invalidCommandMessage, nil
	}
}

func handleExtract(args []string, netlog *NetLog) (string, error) {
	out := ""
	switch args[0] {
	case "dns":
		for _, x := range netlog.FindDNSQueries() {
			out += fmt.Sprintln(x)
		}
		return out, nil
	case "url":
		for _, x := range netlog.FindURLRequests() {
			out += fmt.Sprintln(x)
		}
		return out, nil
	case "red":
		for _, x := range netlog.FindRedirections() {
			out += fmt.Sprintln(x)
		}
		return out, nil
	case "con":
		for _, x := range netlog.FindOpenedSocket() {
			out += fmt.Sprintln(x)
		}
		return out, nil
	case "src":
		if len(args) < 2 {
			return invalidCommandMessage, nil
		}
		_, err := os.Stat(args[1])
		if err != nil {
			err := os.Mkdir(args[1], os.ModePerm)
			if err != nil {
				return "", err
			}
		}
		fileCount := 0
		res := netlog.FindSources()
		if len(res) < 1 {
			return "No sources to extract were found", nil
		}
		for _, x := range res {
			u, err := url.Parse(x.ResourceName)
			if err != nil {
				fmt.Println(err)
				continue
			}
			parts := strings.Split(u.Path, "/")
			name := parts[len(parts)-1]
			if name == "" {
				name = fmt.Sprintf("index-%d", fileCount)
			}
			_, err = os.Stat(fmt.Sprintf("%s/%s", strings.TrimSuffix(args[1], "/"), u.Hostname()))
			if err != nil {
				os.Mkdir(fmt.Sprintf("%s/%s", strings.TrimSuffix(args[1], "/"), u.Hostname()), os.ModePerm)
			}
			filePath := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(args[1], "/"), u.Hostname(), name)
			if len(filePath) > maxFilenameLength {
				fmt.Printf("Trimming %s name beacuse it is too long\n", filePath)
				filePath = filePath[:maxFilenameLength]
			}
			file, err := os.Create(filePath)
			if err != nil {
				fmt.Printf("File %s was not created: %v\n", filePath, err)
				continue
			}
			out := []byte{}
			for _, part := range x.Base64EncodedBytes {
				decoded, err := base64.StdEncoding.DecodeString(part)
				if err != nil {
					fmt.Printf("Skipped some bytes while writing %s because: %v\n", filePath, err)
					continue
				}
				out = append(out, decoded...)
			}
			file.Write(out)
			file.Close()
			fileCount = fileCount + 1
		}
		return fmt.Sprintf("Wrote %d out of %d files\n", fileCount, len(res)), nil
	case helpCase:
		return extractCommandHelpMessage, nil
	default:
		return invalidCommandMessage, nil
	}
}
