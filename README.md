# Apache Exporter for Prometheus

![image](https://user-images.githubusercontent.com/7112075/30365549-acc919e0-989a-11e7-9c31-b9b7e9a5b036.png)

This is a simple server that periodically scrapes apache stats and exports them via HTTP for Prometheus
consumption.

## Requirement

- [Enable Apache server-status](https://unix.stackexchange.com/questions/153915/enable-server-status-on-my-web-server)

```toml
# Server Status Location Config
<Location /server-status>
   SetHandler server-status
   Order deny,allow
   Deny from all
   Allow from all
</Location>
```

> A whole httpd config at: [httpd.conf](./httpd.conf)

## Usage

```bash
$ ./apache_exporter -h

Usage of ./apache_exporter:
  -insecure
     Ignore server certificate if using https (default true)
  -log.level value
     Only log messages with the given severity or above.
 Valid levels: [debug, info, warn, error, fatal, panic].
  -scrape_uri string
     URI to apache server status page. (default "http://localhost/server-status")
  -telemetry.address string
     Address on which to expose metrics. (default ":9113")
  -telemetry.endpoint string
     Path under which to expose metrics. (default "/metrics")
```

> e.g:

```
./apache_exporter -scrape_uri http://localhost/server-status/ -telemetry.address :9113
```

## Metrics

| Metric         | Type  | Descriptions  |
|:------------------|:------|:--------------|
| apache_cpu_load | gauge | CPU Load in % |
| apache_cpu_usage_system | gauge | CPU Usage (System) |
| apache_cpu_usage_user | gauge | CPU Usage (User) |
| apache_data_per_request | gauge | Data per request |
| apache_data_per_second | gauge | Data per second |
| apache_idle_workers | gauge | Idle Workers |
| apache_busy_workers | gauge | Busy Workers |
| apache_number_of_requests_from_client | gauge | Number of requests from client |
| apache_request_currently_being_processed | gauge | Request Currently Being Processed |
| apache_requests_per_second | gauge | Requests per second |
| apache_total_accesses | gauge | Total Accesses |
| apache_total_requests | gauge | Total no of Requests |
| apache_total_traffic | gauge | Total Traffic |
| apache_uptime_days | gauge | Apache server uptime in days |
| apache_uptime_hours | gauge | Apache server uptime hour, but uptime days should be countable |
| apache_uptime_minutes | gauge | Apache server uptime minutes, but uptime days should be countable |
| apache_uptime_seconds | gauge | Apache server uptime seconds, but uptime days should be countable |
| apache_version | gauge | Apache version |
| apache_virtual_hosts | gauge | Number of virtual hosts |
| apache_total_kbytes | gauge | Apache data traffic in KB |
| apache_server_uptime | gauge | Apache httpd server uptime |
| apache_connection_kbytes | gauge | Kilobytes transferred of this connection |
| apache_request_time | gauge | Milliseconds required to process most recent request |
| apache_total_processes | gauge | Apache httpd process number |
