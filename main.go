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

var alive = true

func readinessProbe(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "UP")
}

func livenessProbe(w http.ResponseWriter, req *http.Request) {
	if !alive {
		http.Error(w, "DOWN", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintf(w, "UP")
}

func setColor(w http.ResponseWriter, req *http.Request) {
	color := req.URL.Query().Get("color")
	if color == "" {
		http.Error(w, "Missing parameter: color", http.StatusBadRequest)
		return
	}

	log.Printf("Setting color %s", color)
	if err := runBlink1Tool("--rgb", color); err != nil {
		msg := fmt.Sprintf("Failed to set color %s", color)
		http.Error(w, msg, http.StatusInternalServerError)
		alive = false
		return
	}
	fmt.Fprintf(w, "OK")
}

func turnOff(w http.ResponseWriter, req *http.Request) {
	log.Printf("Turning off LED")
	if err := runBlink1Tool("--off"); err != nil {
		http.Error(w, "Failed to turn off LED", http.StatusInternalServerError)
		alive = false
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

	times := 5
	timesStr := req.URL.Query().Get("times")
	if timesStr != "" {
		newTimes, err := strconv.Atoi(timesStr)
		if err != nil {
			http.Error(w, "Invalid parameter: times", http.StatusBadRequest)
			return
		}
		times = newTimes
	}

	log.Printf("Blinking color %s %d times", color, times)
	if err := runBlink1Tool("--rgb", color, "--blink", strconv.Itoa(times)); err != nil {
		msg := fmt.Sprintf("Failed to blink color %s", color)
		http.Error(w, msg, http.StatusInternalServerError)
		alive = false
		return
	}
	fmt.Fprintf(w, "OK")
}

func runBlink1Tool(params ...string) error {
	cli := "blink1-tool"

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current dir:", err)
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
			log.Fatalf("Failed to parse env variable PORT:", err)
			return
		}
	}

	http.HandleFunc("/turnoff", turnOff)
	http.HandleFunc("/set", setColor)
	http.HandleFunc("/blink", blink)
	http.HandleFunc("/readyz", readinessProbe)
	http.HandleFunc("/livez", livenessProbe)

	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
