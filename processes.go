package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Returns a list of PIDs of running processes with the given name.
// If no processes are found, it returns an empty slice.
// If an error occurs while checking the processes, it returns an error.
func getRunningProcessPids(processName string) ([]string, error) {
	cmd := exec.Command("pidof", processName)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []string{}, nil
		}
		return []string{}, err
	}
	if strings.TrimSpace(string(output)) != "" {
		pids := strings.Fields(string(output))
		return pids, nil
	}
	return []string{}, nil
}

// Starts a process with it's own GPID.
//
// If Config.Constants.DiscardProcessLogs is true, it will also pipe the inputs and outputs of the process into /dev/null
//
// Returns the PID of the detached process as the first return value, and any error as the second. If there is an error, PID will be -1
func runDetachedProcess(command ...string) (int, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if Config.Constants.DiscardProcessLogs {
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			log.Printf("Warning: Could not open /dev/null for detaching process I/O: %v", err)
		} else {
			cmd.Stdin = devNull
			cmd.Stdout = devNull
			cmd.Stderr = devNull
			defer devNull.Close()
		}
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	if err != nil {
		return -1, fmt.Errorf("error starting detached process: %v", err)
	}
	log.Printf("Detached process started with PID: %d", cmd.Process.Pid)
	return cmd.Process.Pid, nil
}

// Tries to kill any running processes with the given name.
// Returns an error if it fails to kill the processes. Returns nil if no processes were found or killed successfully.
func tryKillProcesses(processName string) error {
	runningPids, err := getRunningProcessPids(processName)
	if err != nil {
		return fmt.Errorf("failed to check running processes: %v", err)
	}

	if len(runningPids) > 0 {
		log.Printf("%s is already running, killing old process(es)...", processName)
		var anyErr error
		for _, pid := range runningPids {
			pidInt, err := strconv.Atoi(pid)
			if err != nil {
				log.Printf("Error converting PID '%s' to int: %v", pid, err)
				anyErr = err
				break
			}
			err = syscall.Kill(pidInt, syscall.SIGTERM)
			if err != nil {
				log.Printf("Error killing process with PID %d: %v", pidInt, err)
				anyErr = err
				break
			} else {
				log.Printf("Successfully killed process with PID %d", pidInt)
				continue
			}
		}
		if anyErr != nil {
			return fmt.Errorf("failed to kill one or more processes: %v", anyErr)
		}
		log.Printf("All running processes for %s have been killed", processName)
		return nil
	}

	log.Printf("No running processes found for %s", processName)
	return nil
}
