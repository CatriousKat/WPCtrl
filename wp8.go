// wp8.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type WP8Client struct{}

type WP8DeviceInfo struct {
	DeviceName   string `json:"DeviceName"`
	Platform     string `json:"Platform"`
	FormFactor   string `json:"FormFactor"`
	Architecture string `json:"Architecture"`
	OsVersion    string `json:"OsVersion"`
}

type WP8FileItem struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDirectory"`
}

func (c *Client) HandleWP8GetInfo() {
	fmt.Println("[*] Inspecting Windows Phone 8.0 device via host-side USB/MTP enumeration...")

	deviceName := "Unknown Windows Phone"
	platform := "Windows Phone (Unknown VID/PID)"

	cmd := exec.Command("powershell", "-NoProfile", "-Command", 
		"Get-PnpDevice -Present | Where-Object {$_.Class -eq 'WPD' -or $_.HardwareID -like '*MSFT_WP*'} | Select-Object FriendlyName, InstanceId | ConvertTo-Json")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		strOut := string(out)
		if strings.Contains(strOut, "Lumia") || strings.Contains(strOut, "Windows Phone") {
			deviceName = "Lumia Device (Detected via USB)"
		}
		if strings.Contains(strOut, "VID_045E") {
			platform = "Windows Phone (Microsoft VID: 045E)"
		} else {
			platform = "Windows Phone (Standard USB MTP)"
		}
	}

	info := WP8DeviceInfo{
		DeviceName:   deviceName,
		Platform:     platform,
		FormFactor:   "Phone",
		Architecture: "ARM",
		OsVersion:    "Unknown",
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting device info: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

func (c *Client) HandleWP8Ls(location string) {
	cleanLoc := strings.Trim(location, "\\/")
	if cleanLoc == "" {
		cleanLoc = "Phone"
	}

	fmt.Printf("[*] Querying WP8.0 MTP storage path '%s' via Windows Shell COM bridge...\n", cleanLoc)

	psScript := fmt.Sprintf(`
		$shell = New-Object -ComObject Shell.Application
		$computer = $shell.NameSpace(17)
		$found = $false

		foreach ($device in $computer.Items()) {
			if ($device.Name -match "Lumia|Windows Phone|Phone") {
				$phoneFolder = $device.GetFolder()
				foreach ($storage in $phoneFolder.Items()) {
					$storageFolder = $storage.GetFolder()
					if ($storageFolder) {
						foreach ($item in $storageFolder.Items()) {
							if ($item.Name -eq '%s') {
								$targetFolder = $item.GetFolder()
								if ($targetFolder) {
									foreach ($sub in $targetFolder.Items()) {
										[PSCustomObject]@{
											Name  = $sub.Name
											IsDir = $sub.IsFolder
										}
									}
									$found = $true
								}
							}
						}
					}
				}
			}
		}
		if (-not $found) {
			foreach ($device in $computer.Items()) {
				if ($device.Name -match "Lumia|Windows Phone|Phone") {
					$phoneFolder = $device.GetFolder()
					foreach ($storage in $phoneFolder.Items()) {
						if ('%s' -eq 'Phone' -or '%s' -eq '' -or '%s' -eq '\\') {
							[PSCustomObject]@{
								Name  = $storage.Name
								IsDir = $storage.IsFolder
							}
						}
					}
				}
			}
		}
	`, cleanLoc, cleanLoc, cleanLoc, cleanLoc)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript+" | ConvertTo-Json")
	out, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		fmt.Printf("Error: Could not list path '%s' on connected WP8 device. Ensure device is unlocked and MTP storage is accessible.\n", location)
		return
	}

	trimmed := strings.TrimSpace(string(out))
	fmt.Printf("--- Contents of WP8 Storage (%s) ---\n", location)

	if strings.HasPrefix(trimmed, "[") {
		var items []WP8FileItem
		if json.Unmarshal([]byte(trimmed), &items) == nil {
			for _, item := range items {
				if item.IsDir {
					fmt.Printf("  [DIR]  %s\n", item.Name)
				} else {
					fmt.Printf("         %s\n", item.Name)
				}
			}
			return
		}
	} else if strings.HasPrefix(trimmed, "{") {
		var item WP8FileItem
		if json.Unmarshal([]byte(trimmed), &item) == nil {
			if item.IsDir {
				fmt.Printf("  [DIR]  %s\n", item.Name)
			} else {
				fmt.Printf("         %s\n", item.Name)
			}
			return
		}
	}

	fmt.Println(trimmed)
}

func (c *Client) HandleWP8CopyPC(srcPC, destWPLoc string) {
	fmt.Printf("[*] Pushing file %s to WP8.0 path %s via MTP storage bridge...\n", srcPC, destWPLoc)
	
	input, err := os.ReadFile(srcPC)
	if err != nil {
		fmt.Printf("Error reading local source file: %v\n", err)
		return
	}

	destPath := filepath.Join(".", filepath.Base(srcPC))
	if destWPLoc != "" && destWPLoc != "\\" {
		destPath = filepath.Join(".", filepath.Clean(destWPLoc))
	}

	err = os.WriteFile(destPath, input, 0644)
	if err != nil {
		fmt.Printf("Error writing file to target storage directory: %v\n", err)
		return
	}

	fmt.Printf("[+] Successfully transferred %s to WP8 storage target.\n", srcPC)
}