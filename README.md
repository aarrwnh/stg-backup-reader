https://addons.mozilla.org/firefox/addon/simple-tab-groups/


## run
```
go run . -p $env:DESKTOP/tab-backup
```

## with websockets
```
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 100 -nodes

go run . -p $env:DESKTOP/tab-backup -ws -key key.pem -cert cert.pem
```
