package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DeviceInfo struct {
	OsVersion string `json:"OsVersion"`
}

type AppPackage struct {
	Name            string `json:"Name"`
	PackageFullName string `json:"PackageFullName"`
	AppID           string `json:"AppId"`
}

type ProcessInfo struct {
	Name            string `json:"Name"`
	ProcessId       int    `json:"ProcessId"`
	PackageFullName string `json:"PackageFullName"`
}

type FileItem struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDirectory"`
	Size  int64  `json:"Size"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: "http://169.254.2.2:55755",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) VerifyDeviceConnectionAndVersion(activeCmd string) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/control/deviceinfo")
	if err != nil {
		fmt.Println("Error: No Windows Phone device detected or device is unplugged.")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Device connected, but communication failed.")
		os.Exit(1)
	}

	var info DeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		info.OsVersion = "10.0"
	}

	if strings.HasPrefix(info.OsVersion, "7.") {
		fmt.Printf("Error: Unsupported OS version (%s). Windows Phone 7 devices are not supported.\n", info.OsVersion)
		os.Exit(1)
	}

	// Restrict only features that rely strictly on Windows 10 Mobile Device Portal subsystems
	requiresW10M := map[string]bool{
		"screenshot": true, "list-apps": true, "uninstall": true,
		"launch": true, "terminate": true, "list-processes": true, "kill": true, "install": true,
	}
	
	if requiresW10M[activeCmd] && !strings.HasPrefix(info.OsVersion, "10.") {
		fmt.Printf("Error: The '%s' command requires Windows 10 Mobile. Connected device is running version %s.\n", activeCmd, info.OsVersion)
		os.Exit(1)
	}
}

func (c *Client) HandlePing() {
	fmt.Printf("Pinging Windows Phone RNDIS interface at %s...\n", c.BaseURL)
	successCount := 0

	for i := 1; i <= 4; i++ {
		start := time.Now()
		resp, err := c.HTTPClient.Get(c.BaseURL + "/api/control/deviceinfo")
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("Request %d: Timed out or unreachable (%v)\n", i, err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("Reply from WDP: status=%d time=%v\n", resp.StatusCode, duration)
				successCount++
			} else {
				fmt.Printf("Reply from WDP: unexpected HTTP status=%d time=%v\n", resp.StatusCode, duration)
			}
		}

		if i < 4 {
			time.Sleep(1 * time.Second)
		}
	}

	if successCount == 0 {
		fmt.Println("\nPing statistics: 4 packets transmitted, 0 received (100% packet loss)")
	} else {
		fmt.Printf("\nPing statistics: 4 packets transmitted, %d received\n", successCount)
	}
}

func (c *Client) HandleGetInfo() {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/control/deviceinfo")
	if err != nil {
		fmt.Printf("Error fetching device info: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func (c *Client) HandleInstall(appxPath string) {
	file, err := os.Open(appxPath)
	if err != nil {
		fmt.Printf("Error opening app file: %v\n", err)
		return
	}
	defer file.Close()

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	filename := filepath.Base(appxPath)
	fileWriter, err := bodyWriter.CreateFormFile("package", filename)
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		return
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		fmt.Printf("Error copying file stream: %v\n", err)
		return
	}
	bodyWriter.Close()

	url := fmt.Sprintf("%s/api/app/packagemanager/packages", c.BaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bodyBuf)
	if err != nil {
		fmt.Printf("Error building install request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())

	fmt.Printf("[*] Deploying package %s to device...\n", filename)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing installation request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		fmt.Printf("[+] Successfully installed package: %s\n", filename)
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: Installation failed with status code %d: %s\n", resp.StatusCode, string(bodyBytes))
	}
}

func (c *Client) HandleListApps() {
	pkgResp, err := c.HTTPClient.Get(c.BaseURL + "/api/app/packagemanager/packages")
	if err != nil {
		fmt.Printf("Error fetching packages: %v\n", err)
		return
	}
	defer pkgResp.Body.Close()
	
	var pkgData struct {
		Packages []AppPackage `json:"Packages"`
	}
	if err := json.NewDecoder(pkgResp.Body).Decode(&pkgData); err != nil {
		fmt.Println("Error parsing application package list.")
		return
	}

	procResp, err := c.HTTPClient.Get(c.BaseURL + "/api/taskmanager/processes")
	activePackages := make(map[string]bool)
	if err == nil {
		defer procResp.Body.Close()
		var procData struct {
			Processes []ProcessInfo `json:"Processes"`
		}
		if json.NewDecoder(procResp.Body).Decode(&procData) == nil {
			for _, proc := range procData.Processes {
				if proc.PackageFullName != "" {
					activePackages[proc.PackageFullName] = true
				}
			}
		}
	}

	fmt.Println("--- Installed Applications ---")
	for _, pkg := range pkgData.Packages {
		fmt.Printf("- Name:     %s\n", pkg.Name)
		fmt.Printf("  FullName: %s\n", pkg.PackageFullName)
		if pkg.AppID != "" {
			fmt.Printf("  AUMID:    %s\n", pkg.AppID)
		} else {
			fmt.Printf("  AUMID:    (Unavailable)\n")
		}
		
		if activePackages[pkg.PackageFullName] {
			fmt.Printf("  Status:   Running\n")
		} else {
			fmt.Printf("  Status:   Not running\n")
		}
		fmt.Println()
	}
}

func (c *Client) HandleUninstall(packageFullName string) {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/app/packagemanager/packages?packageFullName=%s", c.BaseURL, packageFullName), nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Error uninstalling package: %v\n", err)
		return
	}
	fmt.Printf("[+] Successfully uninstalled: %s\n", packageFullName)
}

func (c *Client) HandleLaunch(aumid string) {
	resp, err := c.HTTPClient.Post(fmt.Sprintf("%s/api/taskmanager/app?appid=%s", c.BaseURL, aumid), "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Error launching application: %v\n", err)
		return
	}
	fmt.Printf("[+] Launched application: %s\n", aumid)
}

func (c *Client) HandleTerminate(aumid string) {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/taskmanager/app?packageFamilyName=%s", c.BaseURL, aumid), nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Error terminating application: %v\n", err)
		return
	}
	fmt.Printf("[+] Terminated application: %s\n", aumid)
}

func (c *Client) HandleListProcesses() {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/taskmanager/processes")
	if err != nil {
		fmt.Printf("Error fetching processes: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func (c *Client) HandleKill(pid string) {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/taskmanager/processes?pid=%s", c.BaseURL, pid), nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Error killing process ID %s: %v\n", pid, err)
		return
	}
	fmt.Printf("[+] Killed process ID: %s\n", pid)
}

func (c *Client) HandleBattery() {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/control/power/battery")
	if err != nil {
		fmt.Printf("Error fetching battery status: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func (c *Client) HandleLs(location string) {
	url := fmt.Sprintf("%s/api/filemanager/items?knownFolderId=LocalAppData&path=%s", c.BaseURL, location)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		fmt.Printf("Error listing directory: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Directory not found or access denied (HTTP %d)\n", resp.StatusCode)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var rawItems []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawItems); err == nil {
		fmt.Printf("--- Contents of %s ---\n", location)
		for _, item := range rawItems {
			name, _ := item["Name"].(string)
			isDir, _ := item["IsDirectory"].(bool)
			if isDir {
				fmt.Printf("  [DIR]  %s\n", name)
			} else {
				fmt.Printf("         %s\n", name)
			}
		}
		return
	}

	var fileData struct {
		Items []FileItem `json:"Items"`
	}
	if err := json.Unmarshal(bodyBytes, &fileData); err == nil {
		fmt.Printf("--- Contents of %s ---\n", location)
		for _, item := range fileData.Items {
			if item.IsDir {
				fmt.Printf("  [DIR]  %s\n", item.Name)
			} else {
				fmt.Printf("         %s\n", item.Name)
			}
		}
		return
	}

	fmt.Println(string(bodyBytes))
}

func (c *Client) HandleScreenshot(outputPath string) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/screen/screenshot")
	if err != nil {
		fmt.Printf("Error capturing screenshot: %v\n", err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating local file: %v\n", err)
		return
	}
	defer out.Close()

	io.Copy(out, resp.Body)
	fmt.Printf("[+] Screenshot saved to %s\n", outputPath)
}

func (c *Client) HandleReboot() {
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/control/power/reboot", "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Error issuing reboot command.")
		return
	}
	fmt.Println("[+] Device is restarting...")
}

func (c *Client) HandleCopyPC(srcPC, destWPLoc string) {
	file, err := os.Open(srcPC)
	if err != nil {
		fmt.Printf("Error opening local source file: %v\n", err)
		return
	}
	defer file.Close()

	url := fmt.Sprintf("%s/api/filemanager/file?knownFolderId=LocalAppData&path=%s", c.BaseURL, destWPLoc)
	req, err := http.NewRequest(http.MethodPut, url, file)
	if err != nil {
		fmt.Printf("Error creating upload request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	fmt.Printf("[*] Pushing %s to Windows Phone at %s...\n", srcPC, destWPLoc)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("Error uploading file: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		fmt.Printf("[+] Successfully copied %s to phone path %s\n", srcPC, destWPLoc)
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: File transfer rejected with status %d: %s\n", resp.StatusCode, string(bodyBytes))
	}
}

func (c *Client) HandleCopyMobile(srcWP, destPC string) {
	url := fmt.Sprintf("%s/api/filemanager/file?knownFolderId=LocalAppData&path=%s", c.BaseURL, srcWP)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		fmt.Printf("Error pulling file from phone: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: File not found or access denied on device (HTTP %d)\n", resp.StatusCode)
		return
	}

	out, err := os.Create(destPC)
	if err != nil {
		fmt.Printf("Error creating local destination file: %v\n", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("Error writing local file data: %v\n", err)
		return
	}

	fmt.Printf("[+] Successfully pulled %s from phone to local path %s\n", srcWP, destPC)
}