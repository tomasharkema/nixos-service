package main

// curl --unix-socket /run/nixos-service.sock http://localhost -d "derp"

import (
	"bytes"
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

	"github.com/alecthomas/kingpin/v2"
	"github.com/coreos/go-systemd/activation"
	"github.com/coreos/go-systemd/daemon"

	"github.com/godbus/dbus/v5"
)

var (
	pathChannel = make(chan string, 300)

	app = kingpin.New("chat", "A command-line chat application.")

	sockPath = app.Flag("socket", "Channel to post to.").Short('s').Envar("NIXOS_SERVICE_SOCK_PATH").Required().String()

	socketCommand = app.Command("socket", "run as socket")

	atticName = "nixos-service"

	atticServerName = socketCommand.Flag("attic-server-name", "Attic server name").Envar("NIXOS_SERVICE_ATTIC_SERVER_NAME").Required().String()
	atticUrl        = socketCommand.Flag("attic-url", "Attic url").Envar("NIXOS_SERVICE_ATTIC_URL").Required().String()
	atticSecretPath = socketCommand.Flag("attic-secret-path", "Attic name").Envar("NIXOS_SERVICE_ATTIC_SECRET_PATH").Required().String()

	uploadCommand = app.Command("upload", "run as socket")
	uploadNixPath = uploadCommand.Arg("path", "path").Required().String()
)

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

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case socketCommand.FullCommand():
		runSocket(ctx)

	case uploadCommand.FullCommand():
		uploadToSocket(ctx)

	}
}

func runSocket(ctx context.Context) {

	go pollUnits(ctx)
	// go watchdog(ctx)
	go handleSocket(ctx)
	go uploadPath(ctx)

	daemon.SdNotify(false, daemon.SdNotifyReady)

	<-ctx.Done()
	os.Remove(*sockPath)
	log.Println("Exit...")
}

func uploadToSocket(ctx context.Context) {
	fmt.Println("Upload", *uploadNixPath)

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", *sockPath)
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost", strings.NewReader(*uploadNixPath))
	if err != nil {
		panic(err)
	}

	res, err := httpc.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}

func handleSocket(ctx context.Context) {
	// unixListener, err := net.Listen("unix", *sockPath)
	// if err != nil {
	// 	log.Fatal(err)
	// }

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

	listeners, err := activation.Listeners() // ❶
	if err != nil {
		log.Panicf("cannot retrieve listeners: %s", err)
	}
	if len(listeners) != 1 {
		log.Panicf("unexpected number of socket activation (%d != 1)",
			len(listeners))
	}

	err = server.Serve(listeners[0])
	if err != nil {
		panic(err)
	}
}

func uploadPath(ctx context.Context) {

	atticSecret, err := os.ReadFile(*atticSecretPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("login", atticName, *atticUrl, string(atticSecret))
	cmd := exec.Command("attic", "login", atticName, *atticUrl, string(atticSecret))
	cmd.Env = os.Environ()
	res, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(res))

	for {
		path := <-pathChannel

		fmt.Println("Path", path)

		var stderrBuf bytes.Buffer
		fmt.Println("push", fmt.Sprintf("%s:%s", atticName, *atticServerName), "-j1", path)
		cmd := exec.Command("attic", "push", fmt.Sprintf("%s:%s", atticName, *atticServerName), "-j1", path)
		cmd.Stderr = &stderrBuf
		cmd.Env = os.Environ()
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("Error:", err, stderrBuf.String())
		}

		fmt.Println("Output: ", string(out), cmd.ProcessState)
	}
}

func pollUnits(ctx context.Context) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}

	rules := []string{
		"type='signal',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='method_call',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='method_return',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='error',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
	}
	var flag uint = 0

	call := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
	if call.Err != nil {
		panic(call.Err)
	}

	c := make(chan *dbus.Message, 10)
	conn.Eavesdrop(c)

	fmt.Println("Monitoring notifications")

	for v := range c {

		fmt.Println("[dbus]", v)
	}
}

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
