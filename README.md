## SocketIO v4 and EngineIO v4 server implementation for Go

> Note: implementation is not feature-complete.
For example only default namespace is supported.

This repo provides server implementation for SocketIO v4 and EngineIO v4.

Implementation provided is used in [PleaseTalk](https://pleasetalk.app) project.


> Note: Only websocket is supported as transport.
```javascript
const socket = io("https://example.com", { transports: ["websocket"] });
```

### Usage

Basic example

```go
package main

import (
    "net/http"

    "github.com/ffenix113/go-socketio"
    "github.com/ffenix113/go-socketio/engineio"
    "github.com/gobwas/ws/wsutil"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    // This library provides some useful metrics.
    // But if necessary - nil may be passed as registerer.
    reg prometheus.Registerer = nil

    pingInterval = 60 * time.Second
    pingTimeout = 5 * time.Second
    // codec is specifically extracted to show that nil is valid.
    // In this case JSON codec will be used.
    codec socketio.DataCodec = nil
)

func main() {
    sIO := socketio.NewEngine(reg, pingInterval, pingTimeout, wsutil.ReadClientText, wsutil.WriteServerText, codec)

    // This method will be executed when new client connects.
    // The `data` argument is raw `auth` option value as specified [here](https://socket.io/docs/v4/client-options/#auth)
    sIO.OnConnect = func(s *socketio.Socket, _ string, data []byte) (any, error) {
        // Do some validations / JWT parsing for example
        // Then if you want to attach some userID to this socket do
        s.UserID = ""

        return nil, nil
    }

    // Clean-up can be done in `OnDisconnect` callback.
    // At this point socket is already closed, so don't send anything to it.
    sIO.OnDisconnect = func(s *socketio.Socket, event string, data []byte) (any, error) {
		return nil, nil
	}

    // Events can be attached with `sIO.On(..., ...)`.
    // `data` argument contains raw JSON message as sent by the client.
    //
    // Returned value will be passed on to client only if client does [emitWithAck](https://socket.io/docs/v4/client-api/#socketemitwithackeventname-args).
    // If client just does `emit` - response is omitted.
    sIO.On("hello", func(s *socketio.Socket, _ string, data []byte) (any, error) {
        var req struct {
            Name string `json:"name"`
        }

        if err := json.Unmarshal(data, &req); err != nil {
            return nil, err
        }

        // Anything that can be marshaled to JSON can be returned.
        return map[string]string{
            "msg": fmt.Sprintf("hello user %q with id %q", req.Name, s.UserID)
        }, nil
    })

    // Attach socketio handler to any http server
    socketHandler := WebsocketHandler(sIO)
    http.DefaultServeMux.Handle("/socket.io/", socketHandler)
    http.ListenAndServe("0.0.0.0:3000", http.DefaultServeMux)
}

func WebsocketHandler(e *socketio.Engine) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		if query.Get("EIO") != engineio.Version {
			http.Error(rw, "unsupported version", http.StatusBadRequest)
			return
		}
        // Polling is indeed not supported currently.
		if query.Get("transport") == "polling" {
			http.Error(rw, "polling is not supported", http.StatusBadRequest)
			return
		}
        // This is optional, but if client provides `sid` it might expect
        // continuation of the session, which we will not do.
		if sid := query.Get("sid"); sid != "" {
			http.Error(rw, "sid should be empty", http.StatusBadRequest)
			return
		}

		conn, _, _, err := ws.UpgradeHTTP(req, rw)
		if err != nil {
			panic(err)
		}

		e.AddClient(conn)
	}
}
```