package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/charmbracelet/lipgloss"
    "io"
    "os"
    "os/exec"
    "path"
    "strings"
)

type Connection struct {
    Remote_env_path            string   `json:"remote_env_path"`
    Ssh_host                   string   `json:"ssh_host"`
    Ssh_pass                   string   `json:"ssh_pass"`
    Enabled                    bool     `json:"enabled"`
    Local_path                 string   `json:"local_path"`
    Enable_set_gtid_purged_off bool     `json:"enable_set_gtid_purged_off"`
    Id                         string   `json:"id"`
    Ignore_Tables              []string `json:"ignore_tables"`
    With_Core_Config_Data      bool     `json:"with_core_config"`
    Only_Core_Config_Data      bool     `json:"only_core_config"`
}

func main() {
    file, err := os.ReadFile("dump.json")
    if err != nil {
        fmt.Printf("File error: %v\n", err)
    }

    var config_id string
    if len(os.Args) > 1 {
        config_id = os.Args[1]
    }

    var ConnectionItems []Connection

    err = json.Unmarshal(file, &ConnectionItems)
    if err != nil {
        fmt.Printf("File error: %v\n", err)
    }

    if len(ConnectionItems) == 0 {
        fmt.Println("No connection items found")
    }

    infoRender := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
    skipRender := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
    errorRenderer := lipgloss.NewStyle().Foreground(lipgloss.Color("124"))
    count := 0
    for _, item := range ConnectionItems {
        if config_id != "" && strings.ToLower(config_id) != strings.ToLower(item.Id) {
            continue
        }
        count++
        fmt.Println("Found config with Id: ", item.Id)

        var should_skip bool
        if item.Remote_env_path == "" {
            should_skip = true
            fmt.Println(errorRenderer.Render("No remote_env_path found have to skip"))
        }

        if item.Ssh_host == "" {
            should_skip = true
            fmt.Println(errorRenderer.Render("No ssh_host found have to skip"))
        }

        fmt.Println(fmt.Sprintf("Will connect to: %s", infoRender.Render(item.Ssh_host)))
        fmt.Println(fmt.Sprintf("Magento env.php abs location in server is: %s", infoRender.Render(item.Remote_env_path)))

        if item.Enabled == false || should_skip == true {
            fmt.Println(skipRender.Render("Configuration is Disabled Skiping..."))
            fmt.Println(skipRender.Render("#####"))
            fmt.Println(skipRender.Render("####"))
            fmt.Println(skipRender.Render("###"))
            fmt.Println(skipRender.Render("##"))
            fmt.Println(skipRender.Render("#"))
            continue
        }

        if item.Local_path == "" {
            item.Local_path = "./"
        }

        if item.With_Core_Config_Data == true && item.Only_Core_Config_Data == false {
            item.Ignore_Tables = append(item.Ignore_Tables, "core_config_data")
        }

        filename, err := connectAndGenerateDump2(&item)

        if err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("File name: " + filename)
        _ = scpFile(&item, filename)
    }

    if config_id != "" && count == 0 {
        fmt.Println(fmt.Sprintf("No connection items with id %s found", config_id))
    }
}

func connectAndGenerateDump2(item *Connection) (string, error) {

    var cmd *exec.Cmd
    if item.Ssh_pass == "" {
        cmd = exec.Command("ssh", item.Ssh_host, "/bin/bash", "-s")
    } else {
        cmd = exec.Command("sshpass", "-e", "ssh", item.Ssh_host, "/bin/bash", "-s")
        cmd.Env = append(os.Environ(), fmt.Sprintf("SSHPASS=%s", item.Ssh_pass))
    }

    // Print the command string
    // cmd := exec.Command("ssh", "-v", ssh_host, "whoami && pwd && echo $PATH")
    var sent_db_dump_bash_script = generate_db_dump_bash_script
    if item.Enable_set_gtid_purged_off {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__add__purge__id__off__option__", "--set-gtid-purged=OFF")
    } else {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__add__purge__id__off__option__ ", "")
    }

    var tables_to_ignore []string
    for _, table := range item.Ignore_Tables {
        tables_to_ignore = append(tables_to_ignore, fmt.Sprintf("--ignore-table=$DN.%s ", table))
    }

    if item.Only_Core_Config_Data {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__only__core__config__", "core_config_data")
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__name__", "core_config_data")
    } else {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__only__core__config__ ", "")
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, ".__name__", "")
    }

    if len(tables_to_ignore) > 0 {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__ignore__tables__", strings.Join(tables_to_ignore, " "))
    } else {
        sent_db_dump_bash_script = strings.ReplaceAll(sent_db_dump_bash_script, "__ignore__tables__ ", "")
    }

    cmd.Stdin = strings.NewReader(sent_db_dump_bash_script)
    cmd.Args = append(cmd.Args, item.Remote_env_path)

    cmd_string := cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
    //fmt.Println("Environment Variables", cmd.Env)
    fmt.Println("Command to be executed:", cmd_string)

    // Create a buffer to capture stdout
    var stdout_buffer bytes.Buffer
    // Create a MultiWriter that writes to both os.Stdout and the buffer
    cmd.Stdout = io.MultiWriter(os.Stdout, &stdout_buffer)
    cmd.Stderr = os.Stderr // Continue to show stderr in real-time

    // Run the command
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("SSH command finished with error: %v", err)
    }

    // Convert stdout to a string and find the filename
    output := stdout_buffer.String()
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

func scpFile(item *Connection, filename string) error {
    fmt.Println("Starting SCP transfer...")
    var cmd *exec.Cmd

    local_path := path.Join(item.Local_path, path.Base(filename))
    if item.Ssh_pass != "" {
        cmd = exec.Command("sshpass", "-e", "scp", fmt.Sprintf("%s:%s", item.Ssh_host, filename), local_path)
        cmd.Env = append(os.Environ(), fmt.Sprintf("SSHPASS=%s", item.Ssh_pass))
    } else {
        cmd = exec.Command("scp", fmt.Sprintf("%s:%s", item.Ssh_host, filename), local_path)
    }

    cmdString := cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
    //fmt.Println("Environment Variables", cmd.Env)
    fmt.Println("Command to be executed:", cmdString)

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

const generate_db_dump_bash_script = `
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
echo "Starting dump file generation"
filename="/tmp/db.$DN.__name__.$(date +"%d-%m-%y_%H.%M.%S").$((1 + $RANDOM % 100000)).sql.gz"
mysqldump -h$DH -u$DU -p$DP $DN --single-transaction __ignore__tables__ __add__purge__id__off__option__ __only__core__config__ | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\*/\*/' | gzip >"$filename"
echo "Finished dump file generation"
echo "Generated filename: $filename"
`
