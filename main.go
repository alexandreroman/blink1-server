package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func readinessProbe(w http.ResponseWriter, req *http.Request) {
	if err := runBlink1Tool("--rgbread"); err != nil {
		http.Error(w, "NOT_READY", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintf(w, "READY")
}

func livenessProbe(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "UP")
}

func setColor(w http.ResponseWriter, req *http.Request) {
	color := req.URL.Query().Get("color")
	if color == "" {
		http.Error(w, "Missing parameter: color", http.StatusBadRequest)
		return
	}

	delay := 0
	delayStr := req.URL.Query().Get("delay")
	if delayStr != "" {
		var err error
		delay, err = strconv.Atoi(delayStr)
		if err != nil {
			http.Error(w, "Invalid parameter: delay", http.StatusBadRequest)
			return
		}
	}
	if delay < 0 {
		http.Error(w, fmt.Sprintf("Invalid value for parameter delay: %d", delay), http.StatusBadRequest)
		return
	}

	log.Printf("Setting color %s with delay of %d second(s)", color, delay)
	if delay == 0 {
		if err := runBlink1Tool("--rgb", color); err != nil {
			msg := fmt.Sprintf("Failed to set color %s", color)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
	} else {
		if err := runBlink1Tool("--playpattern", fmt.Sprintf("1,%s,%d,0,#000000,0.1,0", color, delay)); err != nil {
			msg := fmt.Sprintf("Failed to set color %s with delay %d", color, delay)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
	}
	fmt.Fprintf(w, "OK")
}

func turnOff(w http.ResponseWriter, req *http.Request) {
	log.Printf("Turning off LED")
	if err := runBlink1Tool("--off"); err != nil {
		http.Error(w, "Failed to turn off LED", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "OK")
}

func blink(w http.ResponseWriter, req *http.Request) {
	color := req.URL.Query().Get("color")
	if color == "" {
		http.Error(w, "Missing parameter: color", http.StatusBadRequest)
		return
	}

	repeat := 5
	repeatStr := req.URL.Query().Get("repeat")
	if repeatStr != "" {
		var err error
		repeat, err = strconv.Atoi(repeatStr)
		if err != nil {
			http.Error(w, "Invalid parameter: repeat", http.StatusBadRequest)
			return
		}
	}
	if repeat < 1 {
		http.Error(w, fmt.Sprintf("Invalid value for parameter repeat: %d", repeat), http.StatusBadRequest)
		return
	}

	log.Printf("Blinking color %s %d times", color, repeat)
	if err := runBlink1Tool("--rgb", color, "--blink", strconv.Itoa(repeat)); err != nil {
		msg := fmt.Sprintf("Failed to blink color %s", color)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "OK")
}

func runBlink1Tool(params ...string) error {
	cli := "blink1-tool"

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current dir")
		return err
	}

	exe := filepath.Join(currentDir, "vendor", fmt.Sprintf("linux-%s", runtime.GOARCH), cli)
	log.Printf("Running CLI: %s %s", cli, strings.Join(params, " "))
	cmd := exec.Command(exe, params[0:]...)

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start %s: %s", cli, err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				log.Printf("Failed to execute %s: exit code=%d", cli, status.ExitStatus())
				return err
			}
		}
	}
	return nil
}

func main() {
	log.Println("Starting blink1-server")

	portString := os.Getenv("PORT")
	var port int
	if portString == "" {
		port = 8080
	} else {
		var err error
		port, err = strconv.Atoi(portString)
		if err != nil {
			log.Fatalf("Failed to parse env variable PORT: %s", portString)
			return
		}
	}

	http.HandleFunc("/off", turnOff)
	http.HandleFunc("/set", setColor)
	http.HandleFunc("/blink", blink)
	http.HandleFunc("/readyz", readinessProbe)
	http.HandleFunc("/livez", livenessProbe)

	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
