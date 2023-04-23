# syncat
Proof-of-concept of bi-directional communication using nothing but SYN/SYN+ACK and TCP sequences.

## usage
The client will not retry if the server isn't responding when the client started.

### server
This will listen on `192.168.0.100`:`8000` and instruct the client to execute `ls /`:
```sh
syncat -ip 192.168.0.100 server 8000 'ls /'
```

### client
This will connect to the server at `192.168.0.100`:`8000` using the local ip address `192.168.0.101`:
```sh
syncat -ip 192.168.0.101 client 192.168.0.100:8000
```
