package testvulns

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
)

// HTTPBodyLeakExample demonstrates unclosed HTTP response body
func HTTPBodyLeakExample() {
	// VULNERABILITY: HTTP response body not closed (CWE-404)
	resp, err := http.Get("https://api.example.com/data")
	if err != nil {
		return
	}
	// Missing: defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

// FilePermissionExample shows insecure file permissions
func FilePermissionExample() {
	// VULNERABILITY: Insecure file permissions (CWE-276)
	file, _ := os.OpenFile("sensitive.txt", os.O_CREATE|os.O_WRONLY, 0777)
	defer file.Close()
	
	// VULNERABILITY: Creating directory with world-writable permissions
	os.Mkdir("/tmp/insecure", 0777)
}

// WeakRandomExample demonstrates weak randomness
func WeakRandomExample() {
	// VULNERABILITY: Using math/rand for security-sensitive operations (CWE-338)
	token := rand.Intn(999999)
	fmt.Printf("Your OTP is: %06d\n", token)
}

// CommandExecutionExample shows dangerous command execution
func CommandExecutionExample(userInput string) {
	// VULNERABILITY: Command injection via exec.Command (CWE-78)
	cmd := exec.Command("sh", "-c", "echo "+userInput)
	output, _ := cmd.Output()
	fmt.Println(string(output))
}

// TempFileExample shows predictable temp file
func TempFileExample() {
	// VULNERABILITY: Predictable temp file (CWE-377)
	tempFile := "/tmp/app_temp_file.txt"
	ioutil.WriteFile(tempFile, []byte("sensitive data"), 0644)
}

// CORSMisconfiguration shows CORS issues
func CORSMisconfiguration(w http.ResponseWriter, r *http.Request) {
	// VULNERABILITY: CORS wildcard (CWE-942)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}

// UnhandledErrorExample shows poor error handling
func UnhandledErrorExample() {
	// VULNERABILITY: Explicitly ignored error (CWE-391)
	file, _ := os.Open("important.txt")
	defer file.Close()
	
	// VULNERABILITY: Error not checked
	data, _ := ioutil.ReadAll(file)
	_ = data
}

// InformationDisclosure shows stack trace exposure
func InformationDisclosure(w http.ResponseWriter, err error) {
	// VULNERABILITY: Stack trace exposed to user (CWE-209)
	if err != nil {
		// Exposing internal error details
		fmt.Fprintf(w, "Error occurred: %v\nStack trace: %+v", err, err)
	}
}

// TODO: Fix this security issue before production
// FIXME: This authentication bypass needs to be addressed
// XXX: Temporary hack for demo