
```
cat /proc/cpuinfo
cat /proc/meminfo
uname -a
cat /etc/lsb-release
df -h
ps -ef
netstat antp
systemctl list-unit-files --type=service
```

```
mysql --host 127.0.0.1 --port 3306 -uroot -p
```


# cert
san.txt
```
subjectAltName = DNS:isucon.ikasako.com
```

```
openssl genrsa -out server.key 2048
openssl req -out server.csr -key server.key -new
openssl x509 -req -days 365 -signkey server.key -in server.csr -out server.crt -extfile san.txt
```