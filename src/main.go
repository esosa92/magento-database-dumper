package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "strings"
)

type Connection struct {
    Env_path string `json:"env_path"`
    Ssh_host string `json:"ssh_host"`
}

func main() {
    file, err := os.ReadFile("dump.json")
    if err != nil {
        fmt.Printf("File error: %v\n", err)
    }

    var ConnectionItems []Connection

    err = json.Unmarshal(file, &ConnectionItems)
    if err != nil {
        fmt.Printf("File error: %v\n", err)
    }

    if len(ConnectionItems) == 0 {
        fmt.Println("No connection items found")
    }

    for _, item := range ConnectionItems {
        fmt.Printf("Connecting to %s\n", item.Ssh_host)
        fmt.Printf("EnvPath location in server is %s\n", item.Env_path)
        filename, err := connectAndGenerateDump2(item.Env_path, item.Ssh_host)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("File name: " + filename)
        _ = scpFile(item.Ssh_host, filename)
    }
}

func connectAndGenerateDump2(env_path, ssh_host string) (string, error) {
    cmd := exec.Command("ssh", ssh_host, "/bin/bash", "-s")
    // cmd := exec.Command("ssh", "-v", ssh_host, "whoami && pwd && echo $PATH")
    cmd.Stdin = strings.NewReader(generateDbDumpBashScript)
    cmd.Args = append(cmd.Args, env_path)

    fmt.Println("Starting generation of dump file, this may take a few minutes...")

    // Create a buffer to capture stdout
    var stdoutBuf bytes.Buffer
    // Create a MultiWriter that writes to both os.Stdout and the buffer
    cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
    cmd.Stderr = os.Stderr // Continue to show stderr in real-time

    // Run the command
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("SCP command finished with error: %v", err)
    }

    // Convert stdout to a string and find the filename
    output := stdoutBuf.String()
    //fmt.Println("Command output captured:", output)

    // Assuming the filename is on the last line of the output
    var filename string
    lines := strings.Split(output, "\n")
    for _, line := range lines {
        if strings.HasPrefix(strings.TrimSpace(line), "Generated filename:") {
            filename = strings.TrimSpace(strings.TrimPrefix(line, "Generated filename:"))
        }
    }

    if filename == "" {
        return "", fmt.Errorf("failed to capture the generated filename")
    }

    return filename, nil
}

func connectAndGenerateDump(env_path, ssh_host string) (string, error) {
    cmd := exec.Command("ssh", ssh_host, "/bin/bash", "-s")
    //cmd := exec.Command("ssh", "-v", ssh_host, "whoami && pwd && echo $PATH")
    cmd.Stdin = strings.NewReader(generateDbDumpBashScript)
    cmd.Args = append(cmd.Args, env_path)

    fmt.Println("Starting generation of dump file, this may take a few minutes...")

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return "", fmt.Errorf("error creating StdoutPipe: %v", err)
    }
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return "", fmt.Errorf("error creating StderrPipe: %v", err)
    }

    // Start the command
    if err := cmd.Start(); err != nil {
        return "", fmt.Errorf("error starting command: %v", err)
    }

    var filename string

    // Print output in real-time
    go func() {
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            line := scanner.Text()
            fmt.Println("stdout:", line)
            if strings.HasPrefix(line, "Generated filename:") {
                filename = strings.TrimSpace(strings.TrimPrefix(line, "Generated filename:"))
            }
        }
    }()

    go func() {
        scanner := bufio.NewScanner(stderr)
        for scanner.Scan() {
            fmt.Println("stderr:", scanner.Text())
        }
    }()

    // Wait for the command to finish
    if err := cmd.Wait(); err != nil {
        return "", fmt.Errorf("command finished with error: %v", err)
    }

    // Check if the filename was captured
    if filename == "" {
        return "", fmt.Errorf("failed to capture the generated filename")
    }

    fmt.Printf("Dump file generated: %s\n", filename)

    return filename, nil
}

func scpFile(ssh_host, filename string) error {
    fmt.Println("Starting SCP transfer...")

    // Create the SCP command with the verbose flag
    cmd := exec.Command("scp", fmt.Sprintf("%s:%s", ssh_host, filename), "./")

    // Set the command's stdout and stderr to the process's stdout and stderr
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Run the command
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("SCP command finished with error: %v", err)
    }

    fmt.Println("SCP transfer completed successfully")

    return nil
}

var generateDbDumpBashScript = `
#!/bin/bash

ENV_PHP_PATH=$1

#echo "Current user: $(whoami)"
#echo "Current directory: $(pwd)"
#echo "PATH: $PATH"
#echo $ENV_PHP_PATH

DN="$(grep "[\']db[\']" -A 20 "$ENV_PHP_PATH" | grep "dbname" | head -n1 | sed "s/.*[=][>][ ]*[']//" | sed "s/['][,]//")"
DH="$(grep "[\']db[\']" -A 20 "$ENV_PHP_PATH" | grep "host" | head -n1 | sed "s/.*[=][>][ ]*[']//" | sed "s/['][,]//")"
DU="$(grep "[\']db[\']" -A 20 "$ENV_PHP_PATH" | grep "username" | head -n1 | sed "s/.*[=][>][ ]*[']//" | sed "s/['][,]//")"
DP="$(grep "[\']db[\']" -A 20 "$ENV_PHP_PATH" | grep "password" | head -n1 | sed "s/.*[=][>][ ]*[']//" | sed "s/[']$//" | sed "s/['][,]//")"

#echo $DN $DH $DU $DP

filename="/tmp/db.$DN.$(date +"%d-%m-%y_%H.%M.%S").$((1 + $RANDOM % 100000)).sql.gz"
mysqldump -h$DH -u$DU -p$DP $DN --single-transaction --set-gtid-purged=OFF | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\*/\*/' | pv | gzip >"$filename"
echo "Generated filename: $filename"
`
