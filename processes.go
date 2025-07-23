package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

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
		log.Printf("Process '%s' is running with PIDs: %v", processName, pids)
		return pids, nil
	}
	return []string{}, nil
}

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
		log.Printf("Failed to start detached process: %v", err)
		return -1, err
	}
	log.Printf("Detached process started with PID: %d", cmd.Process.Pid)
	return cmd.Process.Pid, nil
}
