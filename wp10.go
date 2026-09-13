// wp10.go
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
)

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