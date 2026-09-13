// wp81.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type DeviceInfo struct {
	OsVersion string `json:"OsVersion"`
}

type FileItem struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDirectory"`
	Size  int64  `json:"Size"`
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

	if len(info.OsVersion) >= 2 && info.OsVersion[0:2] == "7." {
		fmt.Printf("Error: Unsupported OS version (%s). Windows Phone 7 devices are not supported.\n", info.OsVersion)
		os.Exit(1)
	}

	requiresW10M := map[string]bool{
		"screenshot": true, "list-apps": true, "uninstall": true,
		"launch": true, "terminate": true, "list-processes": true, "kill": true, "install": true,
	}

	isW10M := len(info.OsVersion) >= 3 && info.OsVersion[0:3] == "10."
	if requiresW10M[activeCmd] && !isW10M {
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