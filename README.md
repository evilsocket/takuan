Takuan is a system service that parses logs and dectects noisy attackers in order to build a blacklist database of
 known cyber offenders.
 
Periodic reports are saved to [this repository](https://github.com/evilsocket/takuan-reports) in CSV format for
 parsing. Twitter bot running as [@cybertakuan](https://twitter.com/cybertakuan) and tweeting about new reports.
 
## How to Use

Install the configuration:

    sudo mkdir -p /etc/takuan
    sudo cp config.example.yml /etc/takuan/config.yml

Use your favorite editor to customize it to your needs, then you can build and start all the takuan services via
 `docker-compose`:

    sudo docker-compose build
    sudo docker-compose up
   
Reports are saved on the host `/var/log/takuan/reports` and all events are available on a MySQL database running in
 one of the container and persisting its data in `/var/lib/takuan`. A `phpmyadmin` is also available on `http
 ://localhost:9090`.
    
## License

`takuan` is made with ♥  by [evilsocket](https://github.com/evilsocket) and it's released under the GPL 3
 license.