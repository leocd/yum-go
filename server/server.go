package server

import (
        "context"
        "errors"
        "flag"
        "fmt"
        "log"
        "net"
        "net/http"
        "os"
        "strings"
)

func Main() int {
        programName := os.Args[0]
        errorLog := log.New(os.Stderr, "", log.LstdFlags)
        serveLog := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

        flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
        flags.Usage = func() {
                out := flags.Output()
                fmt.Fprintf(out, "Usage: %v [args] [dir]\n\n", programName)
                fmt.Fprint(out, "  dir为需要开启web服务的目录，是可选参数; 如果没有设置, 将使用'.'作为默认值。\n")
                fmt.Fprint(out, "  如果需要指定dir，需要将其作为最后一个参数。\n\n")
                fmt.Fprint(out, "  默认情况下，yum-go将监听0.0.0.0:8080。\n")
                fmt.Fprint(out, "  host(-h)和port(-p)均可使用参数更改。\n")
                fmt.Fprint(out, "  如果需要yum-go监听特定IP，请使用-h参数指定。\n")
                fmt.Fprint(out, "  如果需要yum-go监听指定端口，请使用-p参数指定。\n")
                fmt.Fprint(out, "  如果-p参数设置为0，将随机监听一个未被占用的端口。\n\n")
                flags.PrintDefaults()
        }

        hostFlag := flags.String("h", "0.0.0.0", "指定yum-go需要监听的ip")
        portFlag := flags.String("p", "8080", "指定yum-go需要监听的端口，如果设置为0，将随机监听一个未被占用的端口")
        addrFlag := flags.String("addr", "0.0.0.0:8080", "完整的监听信息(host:port)，请不要与-h或者-p一起使用")

        flags.Parse(os.Args[1:])

        if len(flags.Args()) > 1 {
                errorLog.Println("Error: 输入了太多的参数")
                flags.Usage()
                os.Exit(1)
        }

        rootDir := "."
        if len(flags.Args()) == 1 {
                rootDir = flags.Args()[0]
        }

        allSetFlags := flagsSet(flags)
        if allSetFlags["addr"] && (allSetFlags["host"] || allSetFlags["port"]) {
                errorLog.Println("Error: 使用addr参数时，不可使用h或p参数")
                flags.Usage()
                os.Exit(1)
        }

        var addr string
        if allSetFlags["addr"] {
                addr = *addrFlag
        } else {
                addr = *hostFlag + ":" + *portFlag
        }
        srv := &http.Server{
                Addr: addr,
        }

        // 一个简单的关闭server的方法
        shutdownCh := make(chan struct{})
        go func() {
                <-shutdownCh
                srv.Shutdown(context.Background())
        }()

        mux := http.NewServeMux()
        mux.HandleFunc("/__internal/__shutdown", func(w http.ResponseWriter, r *http.Request) {
                r.ParseForm()
                Key:=r.FormValue("Key")
                if Key != "" && Key == "shutdown_yum_go" {
                        w.WriteHeader(http.StatusOK)
                        defer close(shutdownCh)
                } else {
                        http.Error(w, "403 Forbidden", http.StatusForbidden)
                }
        })

        fileHandler := serveLogger(serveLog, http.FileServer(http.Dir(rootDir)))
        mux.Handle("/", fileHandler)
        srv.Handler = mux

        // 当端口设置为0时，随机分配一个可使用的端口
        listener, err := net.Listen("tcp", addr)
        if err != nil {
                errorLog.Println(err)
                return 1
        }
        scheme := "http://"
        serveLog.Printf("Serving directory %q on %v%v", rootDir, scheme, listener.Addr())

        err = srv.Serve(listener)
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
                errorLog.Println("Error in Serve:", err)
                return 1
        } else {
                return 0
        }
}

func flagsSet(flags *flag.FlagSet) map[string]bool {
        s := make(map[string]bool)
        flags.Visit(func(f *flag.Flag) {
                s[f.Name] = true
        })
        return s
}

func serveLogger(logger *log.Logger, next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                remoteHost, _, _ := strings.Cut(r.RemoteAddr, ":")
                logger.Printf("%v %v %v\n", remoteHost, r.Method, r.URL.Path)
                next.ServeHTTP(w, r)
        })
}
