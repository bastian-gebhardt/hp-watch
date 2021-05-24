package main

import (
    "bufio"
    "flag"
    "fmt"
    "github.com/robfig/cron"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

var  version = "0.0.0"

type node struct {
    name string
    child []*node
    parent *node
    entries []string
}

func main() {
    versionFlag := flag.Bool("version", false, "Shows the version")
    flag.Parse()
    if *versionFlag {
        fmt.Printf("    Version: %s\n", version)
        return
    }

    output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Stamp}
    log.Logger = zerolog.New(output).With().Timestamp().Logger()
    zerolog.SetGlobalLevel(zerolog.InfoLevel)

    c := cron.New()
    c.AddFunc("*/1 * * * *", func() {
        content := getData()
        found, index := findProfile(content)
        switch {
        case strings.Contains(found,"a2dp_sink"):
            log.Debug().Msg("Profile found, nothing to do")
        case found == "":
            log.Debug().Msg("Headset 60_AB_D2_29_7E_F8 is not active, nothing to do")
        case strings.Contains(found, "active profile:"):
            log.Info().Msg("Headset 60_AB_D2_29_7E_F8 is active, but with wrong profile, will switch to 'a2dp_sink'")
            setCorrectProfile(index)
        }
    })
    c.Run()
}

func setCorrectProfile(index int) {
    iStr := strconv.Itoa(index)
    cmd := exec.Command("pacmd", "set-card-profile", iStr, "a2dp_sink")
    //cmdReader, _ := cmd.StdoutPipe()
    err := cmd.Start()
    if err != nil {
        log.Error().Msg("Cannot start command")
        return
    }
    err = cmd.Wait()
    if err != nil {
        log.Error().Msg("Cannot wait for command")
        return
    }
}

// runs 'pacmd list-cards' and returns commands output as slices of lines
func getData() []string {
    var content []string
    cmd := exec.Command("pacmd", "list-cards")
    cmdReader, _ := cmd.StdoutPipe()
    err := cmd.Start()
    if err != nil {
        log.Error().Msg("Cannot start command")
        return nil
    }
    go func() {
        scanner := bufio.NewScanner(cmdReader)
        scanner.Split(bufio.ScanLines)
        for scanner != nil && scanner.Scan() {
            content = append(content, scanner.Text())
        }
    }()

    err = cmd.Wait()
    if err != nil {
        log.Error().Msg("Cannot wait for command")
        return nil
    }

    return content
}

// find line with active profile info for headset in commands 'pacmd list-cards' output
func findProfile(lines []string) (string, int) {
    var err error
    indentation := 0
    searchMode := false
    found := ""
    index := 0

    loop:
    for _, line := range lines {
        //replace tabs with 2 spaces
        line = strings.Replace(line, "\t", "  ", -1)
        if strings.Contains(line,"index:") {
            parts := strings.Split(line, ":")
            index, err = strconv.Atoi(strings.Trim(parts[1]," "))
            if err != nil {
                log.Error().Msgf("Cannot convert index number '%s' to int", parts[1])
            }
            continue
        }
        if ! searchMode {
            if strings.Contains(line,"name: <bluez_card.60_AB_D2_29_7E_F8>") {
                indentation = getIndentation(line)
                searchMode = true
                log.Debug().Msgf("Found headset in line: %s", line)
            }
        } else {
            //
            if indentation > getIndentation(line){
                found = ""
                log.Debug().Msgf("Found indention change in line: %s", line)
                break loop
            }
            if strings.Contains(line, "active profile:") {
                found = line
                break loop
            }
        }
    }

    return found, index
}

func readFile (fileName string) []string {
    file, err := os.Open(fileName)
    if err != nil {
        log.Fatal().Msg("failed to open")
    }
    defer func() {
        err = file.Close()
        if err != nil {
            log.Error().Msg("cannot close file handler")
        }
    }()

    scanner := bufio.NewScanner(file)
    scanner.Split(bufio.ScanLines)
    var text []string

    for scanner.Scan() {
        text = append(text, scanner.Text())
    }

    return text
}


func test () {
    lines := []string {
        "a",
        " b",
        "  c",
        "   d",
        "    e",
        "f",
        "g",
        "\th",
        "\t\ti",
        "\t\t\tk",
        "\t\t\t\tl",
    }
    parsePAOut(lines)
}

func parsePAOut(lines []string) {
    indentation := 0
    lastLine := "root"
    root := newNode(lastLine)
    var keyIndex map[int]*node
    keyIndex = make(map[int]*node)
    keyIndex[indentation] = root
    n := root
    for _, line := range lines {
        //replace tabs with 2 spaces
        line = strings.Replace(line,"\t","  ", -1)
        newIndentation := getIndentation(line)
        upOrDown := newIndentation - indentation
        indentation = newIndentation
        switch {
        case upOrDown < 0:
            // up
            // TODO: check if nil (root node)
            n = n.parent
            n.entries = append(n.entries, strings.Trim(line," "))
        case upOrDown == 0:
            n.entries = append(n.entries, strings.Trim(line," "))
        case upOrDown > 0:
            // down
            keyIndex[indentation] = n
            log.Info().Msgf("goes down: >%v<", line)
            newNode := newNode(lastLine)
            newNode.parent = n
            newNode.entries = append(newNode.entries, strings.Trim(line," "))
            n.child = append(n.child, newNode)
            n = newNode
        }
        lastLine = line
        log.Debug().Msgf("Indent: %v", newIndentation)
    }
    fmt.Println("Ergebnis")
    traverse(root)
}

func newNode(name string) *node {
    n := node{
        name: name,
        child:   nil,
        parent:  nil,
        entries: nil,
    }
    return &n
}

func traverse(n *node) {
    stop := false
    for ! stop {
        fmt.Println(n.entries)
        if n.child != nil {
            n = n.child[0]
        } else {
            stop = true
        }
    }
}

func lineToKeyValue(line string) (string, string) {
    parts := strings.SplitN(line, ":", 1)
    c := len(parts)
    switch {
    case c == 1:
        return strings.Trim(parts[0]," "), ""
    case c == 2:
        return strings.Trim(parts[0]," "), strings.Trim(parts[1]," ")
    }
    return "", ""
}

func getIndentation(l string) int {
    length := len(l)
    trimmed := strings.TrimLeft(l, " ")
    trimmedLength := len(trimmed)
    return length - trimmedLength
}