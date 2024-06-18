package main

// curl --unix-socket /run/nixos-service.sock http://localhost -d "derp"

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/coreos/go-systemd/daemon"
)

var pathChannel = make(chan string, 300)
var sockPath = os.Getenv("NIXOS_SERVICE_SOCK_PATH")

// func HelloServer(w http.ResponseWriter, req *http.Request) {
// 	io.WriteString(w, "hello socket activated world!\n")
// 	fmt.Fprintf(w, "%v", req)
// }

// func main() {
// 	listeners, err := activation.Listeners()
// 	if err != nil {
// 		panic(err)
// 	}

// 	if len(listeners) != 1 {
// 		panic("Unexpected number of socket activation fds")
// 	}

// 	http.HandleFunc("/", HelloServer)
// 	http.Serve(listeners[0], nil)
// }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// go pollUnits(ctx)
	// go watchdog(ctx)
	go handleSocket(ctx)
	go uploadPath(ctx)

	daemon.SdNotify(false, daemon.SdNotifyReady)

	<-ctx.Done()
	os.Remove(sockPath)
	log.Println("Exit...")
}

func handleSocket(ctx context.Context) {
	unixListener, err := net.Listen("unix", sockPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Start socket")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		var strBuf strings.Builder
		_, err := io.Copy(&strBuf, r.Body)
		defer r.Body.Close()

		if err != nil {
			panic(err)
		}

		path := strBuf.String()
		fmt.Println("REQ", path)
		pathChannel <- path

	})

	server := http.Server{
		Handler: mux,
	}

	err = server.Serve(unixListener)
	if err != nil {
		panic(err)
	}
}

func uploadPath(ctx context.Context) {

	atticName := os.Getenv("NIXOS_SERVICE_ATTIC_NAME")
	atticServerName := os.Getenv("NIXOS_SERVICE_ATTIC_SERVER_NAME")
	atticUrl := os.Getenv("NIXOS_SERVICE_ATTIC_URL")
	atticSecretPath := os.Getenv("NIXOS_SERVICE_ATTIC_SECRET_PATH")

	atticSecret, err := os.ReadFile(atticSecretPath)
	if err != nil {
		fmt.Println(err)
	}

	res, err := exec.Command("attic", "login", atticName, atticUrl, string(atticSecret)).Output()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(res))

	for {
		path := <-pathChannel

		fmt.Println("Path", path)

		out, err := exec.Command("attic", "push", atticServerName, "-j1", path).Output()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Output: ", string(out))
	}
}

// func pollUnits(ctx context.Context) {
// 	conn, err := dbus.ConnectSessionBus()
// 	if err != nil {
// 		panic(err)
// 	}

// 	rules := []string{
// 		"type='signal',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
// 		"type='method_call',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
// 		"type='method_return',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
// 		"type='error',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
// 	}
// 	var flag uint = 0

// 	call := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
// 	if call.Err != nil {
// 		panic(call.Err)
// 	}

// 	c := make(chan *dbus.Message, 10)
// 	conn.Eavesdrop(c)

// 	fmt.Println("Monitoring notifications")

// 	for v := range c {

// 		fmt.Println(v)
// 	}
// }

// func watchdog(ctx context.Context) {
// 	interval, err := daemon.SdWatchdogEnabled(false)
// 	if err != nil || interval == 0 {
// 		return
// 	}
// 	for {
// 		req, err := http.NewRequestWithContext(ctx, "get", "http://127.0.0.1:8081", nil)
// 		_, err = http.DefaultClient.Do(req)
// 		if err == nil {
// 			daemon.SdNotify(false, daemon.SdNotifyWatchdog)
// 		}
// 		time.Sleep(interval / 3)
// 	}
// }
