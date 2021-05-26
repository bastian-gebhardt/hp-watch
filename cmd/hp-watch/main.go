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

const programID = "b3507fe0c5c387e072f5f89505c0b03ad35da51f"

// set by compiler
var version = "0.0.0"
var buildDate string

type config struct {
    versionFlag bool
    bluetoothID string
    checkPeriod int
}

// TODO: flag validation
// TODO: fill README.md
// TODO: add tests
// TODO: add linter support
// TODO: add check if 'pacmd' is available
// TODO: add config file support
// TODO: add log file support
// TODO: add parser for 'pacmd' output
// DONE: add flags for bluetooth address and check period
// DONE: remove outdated code (node parsing)

func main() {
    // configure logger
    output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Stamp}
    log.Logger = zerolog.New(output).With().Timestamp().Logger()
    zerolog.SetGlobalLevel(zerolog.InfoLevel)

    cfg := parseCommandLineFlags()

    if cfg.versionFlag {
        printVersionInfo()
        os.Exit(0)
    }

    c := cron.New()
    c.AddFunc(fmt.Sprintf("*/%v * * * *", cfg.checkPeriod), func() {
        pacmdOut := getPacmdOutput()
        found, index := findProfile(pacmdOut, cfg.bluetoothID)
        switch {
        case strings.Contains(found,"a2dp_sink"):
            log.Debug().Msg("Profile found, nothing to do")
        case found == "":
            log.Debug().Msgf("Headset %s is not active, nothing to do", cfg.bluetoothID)
        case strings.Contains(found, "active profile:"):
            log.Info().Msgf("Headset %s is active, but with wrong profile, will switch to 'a2dp_sink'", cfg.bluetoothID)
            setProfileForCard(index, "a2dp_sink")
        }
    })
    log.Info().Msgf("start watching for profile changes at bluetooth device %s every %vs", cfg.bluetoothID, cfg.checkPeriod)
    c.Run()
}

func parseCommandLineFlags() config{
    var cfg config

    versionFlag := flag.Bool("version", false, "Shows the version")
    bluetoothIDFlag := flag.String("id", "", "ID of bluetooth device (Format: XX:XX:XX:XX:XX:XX or XX_XX_XX_XX_XX_XX)")
    checkPeriodFlag := flag.Int("check", 5, "check period in seconds (default: 5)")

    flag.Usage = func() {
        fmt.Printf("Usage of %s:\n", os.Args[0])
        flag.PrintDefaults()
    }
    flag.Parse()

    // validate flags and put adapted values to config
    switch {
    case *checkPeriodFlag<1:
        log.Warn().Msgf("value of flag '--check' is '%v', but must be in range 1-59; autocorrect to: %v", *checkPeriodFlag, 1)
        cfg.checkPeriod = 1
    case *checkPeriodFlag>59:
        log.Warn().Msgf("value of flag '--check' is '%v', but must be in range 1-59; autocorrect to: %v", *checkPeriodFlag, 59)
        cfg.checkPeriod = 59
    default:
        cfg.checkPeriod = *checkPeriodFlag
    }

    // TODO: validation needed
    cfg.bluetoothID = strings.Replace(*bluetoothIDFlag, ":", "_", -1)

    cfg.versionFlag = *versionFlag

    return cfg
}

func printVersionInfo() {
    fmt.Printf("    Version    :    %s\n    Build date :    %s\n    ProgramID  :    %s\n", version, buildDate, programID)
}

// runs pacmd command to set profile
func setProfileForCard(index int, profile string) {
    iStr := strconv.Itoa(index)
    cmd := exec.Command("pacmd", "set-card-profile", iStr, profile)

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
func getPacmdOutput() []string {
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
func findProfile(lines []string, bluetoothID string) (string, int) {
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
            if strings.Contains(line,fmt.Sprintf("name: <bluez_card.%s>", bluetoothID)) {
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

// returns count of spaces before text in line begins
// does not work for tabs
func getIndentation(l string) int {
    length := len(l)
    trimmed := strings.TrimLeft(l, " ")

    return length - len(trimmed)
}