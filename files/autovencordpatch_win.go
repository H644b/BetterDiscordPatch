package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/windows"
)

const (
	checkInterval  = 1 * time.Second
	updateTimeout  = 2 * time.Minute
	updatePollRate = 2 * time.Second
)

func runInstaller() {
	cmd := exec.Command(filepath.Join(os.Getenv("LOCALAPPDATA"), "betterdiscordpatch/betterdiscordpatch.exe"))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	err := cmd.Run()
	if err != nil {
		fmt.Println("["+time.Now().Format("2006-01-02 15:04:05")+"] Failed to run installer:", err)
	}
}

func killDiscord() {
	cmd := exec.Command("C:\\Windows\\System32\\taskkill.exe", "/f", "/im", "Discord.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	err := cmd.Start()
	if err != nil {
		fmt.Println("["+time.Now().Format("2006-01-02 15:04:05")+"] Failed to kill Discord:", err)
	}
}

func startDiscord() {
	cmd := exec.Command(filepath.Join(os.Getenv("LOCALAPPDATA"), "Discord/Update.exe"), "--processStart", "Discord.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	err := cmd.Start()
	if err != nil {
		fmt.Println("["+time.Now().Format("2006-01-02 15:04:05")+"] Failed to start Discord:", err)
	}
}

// waitForUpdateComplete polls the new app directory until core.asar appears in
// the discord_desktop_core module, indicating that Squirrel has finished
// extracting the update. Returns true on success or false if it times out.
func waitForUpdateComplete(appDir string) bool {
	deadline := time.Now().Add(updateTimeout)
	for time.Now().Before(deadline) {
		modulesPath := filepath.Join(appDir, "modules")
		entries, err := os.ReadDir(modulesPath)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "discord_desktop_core") {
					corePath := filepath.Join(modulesPath, entry.Name(), "discord_desktop_core", "core.asar")
					if _, err := os.Stat(corePath); err == nil {
						return true
					}
				}
			}
		}
		time.Sleep(updatePollRate)
	}
	return false
}

func main() {
	discordDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Discord")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("["+time.Now().Format("2006-01-02 15:04:05")+"] Failed to create watcher:", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(discordDir)
	if err != nil {
		fmt.Println("["+time.Now().Format("2006-01-02 15:04:05")+"] Failed to add watcher:", err)
		return
	}

	fmt.Println("[" + time.Now().Format("2006-01-02 15:04:05") + "] Watching for Discord updates...")

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				name := filepath.Base(event.Name)
				if strings.HasPrefix(name, "app-") {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						fmt.Println("[" + time.Now().Format("2006-01-02 15:04:05") + "] Discord is updating (new version: " + name + "), waiting for completion...")
						if waitForUpdateComplete(event.Name) {
							fmt.Println("[" + time.Now().Format("2006-01-02 15:04:05") + "] Discord update complete, running BetterDiscord patcher...")
							killDiscord()
							runInstaller()
							startDiscord()
						} else {
							fmt.Println("[" + time.Now().Format("2006-01-02 15:04:05") + "] Timed out waiting for Discord update to complete")
						}
					}
				}
			}
		case err := <-watcher.Errors:
			fmt.Println("Watcher error:", err)
			time.Sleep(checkInterval)
		}
	}
}
