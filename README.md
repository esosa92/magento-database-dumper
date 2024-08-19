# Magento Database Dumper

With this utility you can configure connection information and magento env.php file absolute path. The tool will cycle through the configs and download automatically or you can send it an specific configuration id to download.

The precompiled binary is part of the repository. You can ofc download and compile yourself.

## How it works

After you get the binary the program tries to search a file name **dump.json** in the same directory. With the following example configuration:

```json
[
    {
        "remote_env_path": "/absolute/path/to/remote/server/env.php",
        "ssh_host": "user@host",
        "local_path": "/absolute/path/where/to/download/dump",
        "enabled": false,
        "enable_set_gtid_purged_off": true,
        "id": "LaCabra"
    },
    {
      "remote_env_path": "/absolute/path/to/remote/server/env.php",
      "local_path": "/absolute/path/where/to/download/dump",
      "ssh_host": "watc_stg",
      "enabled": true,
      "ssh_pass": "SuperDuperSecretPassword"
    }
]
```
- **remove_env_path**: This option is required. Here you set up where the env file is located in the remote server. It can work with relative or absolute. I would recommend an absolute path.
- **ssh_host**: This option is required. Here you can put a host that you have configured in .ssh/config file or a user and host(user@examplehost.com or user@192.36.21.50).
- **local_path**: This is optional if you don't specify anything the program will attempt to download where it was executed. Otherwise it will try to use the path you put here. The path should be a directory and it must exist. It is recommended it is absolute. You don't specify the filename here. Only where you want the resulting file to be downloaded.
- **enabled**: By default is false. If you want this config to work you have to set it to enabled: true. When this config is present with false or not present the configuration will be ignored.
- **enable_set_gtid_purged_off**: sometimes you need this to be present when dumping. In those cases that you want this option to be present you must add the option and set it to "true".
- **ssh_pass**: If you set up this option it will attempt to use the program sshpass. With this program you can effectively add the connection password and not need to be paying attention to the password prompting. If you don't set this up and you have a server which requires an ssh password because it has auth keys disabled, then  you will need to put the password twice. One for the connection and dumping another for the scping and download of the db_dump file.
- **id**: This is an optional custom string and it is used for sending the command the only parameter it accepts which is an id.

To run this where you have the binary then you execute the binary and you can pass the optional **id** parameter. If you pass an id it will attempt to download all the configs that have that ID. If you don't send any ID it will attempt to dump all projects that you have configured. For both things enabled must set to "true".

The filename is automatically generated
`filename="/tmp/db.$DN.$(date +"%d-%m-%y_%H.%M.%S").$((1 + $RANDOM % 100000)).sql.gz"`
if the database_name is example, and you run this dumper the 01/05/2027 at 10:15:23 the resulting name will be
`db.example.2027-01-05_10.15.23.{random_number_between_1_and_100000}`

This is the mysqldump command that will be run
`mysqldump -h$DH -u$DU -p$DP $DN --single-transaction __add__purge__id__off__option__ | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\*/\*/' | pv | gzip >"$filename"`