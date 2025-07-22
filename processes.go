package main

import (
	"log"
	"os/exec"
	"strings"
	"syscall"
)

func getRunningProcessPids(processName string) ([]string, error) {
	cmd := exec.Command("pidof", processName)
	// if error code is 1, the process is not running
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []string{}, nil 
		}
		return []string{}, err 
	}
	// If output is not empty, return the pids
	if strings.TrimSpace(string(output)) != "" {
		pids := strings.Fields(string(output))
		log.Printf("Process '%s' is running with PIDs: %v", processName, pids)
		return pids, nil
	}
	return []string{}, nil
}

func runDetachedProcess(command ...string) (int, error) {
	cmd := exec.Command("sh", "-c", strings.Join(command, " "))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // run in a new process group
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to start detached process: %v", err)
		return -1, err
	}
	log.Printf("Detached process started with PID: %d", cmd.Process.Pid)
	return cmd.Process.Pid, nil
}
