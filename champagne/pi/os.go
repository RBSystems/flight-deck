package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func fixTime() error {
	fmt.Printf("Fixing time\n")

	cmd := exec.Command("ntpdate", "tick.byu.edu")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to fix time: %s", err)
	}

	return nil
}

func updateAndReboot() error {
	data.Lock()
	data.ProgressMessage = "fixing time"
	data.ProgressPercent = 5
	data.Unlock()

	if err := fixTime(); err != nil {
		return err
	}

	data.Lock()
	data.ProgressMessage = "updating apt"
	data.ProgressPercent = 15
	data.Unlock()

	fmt.Printf("Updating apt\n")

	// update apt
	cmd := exec.Command("apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt update", err)
	}

	fmt.Printf("\nUpgrading packages\n")

	data.Lock()
	data.ProgressPercent = 30
	data.ProgressMessage = "upgrading packages"
	data.Unlock()

	// upgrade packages
	cmd = exec.Command("apt", "-y", "upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y upgrade", err)
	}

	fmt.Printf("\nRemoving leftover packages\n")

	data.Lock()
	data.ProgressPercent = 75
	data.ProgressMessage = "removing leftover packages"
	data.Unlock()

	// remove/clean leftover junk
	cmd = exec.Command("apt", "-y", "autoremove")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoremove", err)
	}

	fmt.Printf("\nCleaning apt cache\n")

	data.Lock()
	data.ProgressPercent = 90
	data.ProgressMessage = "cleaning apt cache"
	data.Unlock()

	cmd = exec.Command("apt", "-y", "autoclean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %q: %w", "apt -y autoclean", err)
	}

	fmt.Printf("\n\n\nDone! Rebooting!!\n")
	data.Lock()
	data.ProgressPercent = 99
	data.ProgressMessage = "rebooting"
	data.Unlock()

	time.Sleep(10 * time.Second)
	return reboot()
}

func reboot() error {
	// for more info, look at the man page for reboot(2)
	return unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART)
}

func source(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	env := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			// skip empty lines & comments
			continue
		}

		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			return fmt.Errorf("invalid line found: %q", line)
		}

		env[split[0]] = split[1]
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("unable to scan file: %w", err)
	}

	// actually set the env vars
	for k, v := range env {
		os.Setenv(k, v)
	}

	return nil
}