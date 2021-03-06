package main

import (
    "os"
    "net"
    "time"
    "errors"
    "net/http"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

const (

    // https://www.percona.com/doc/percona-xtradb-cluster/5.7/wsrep-status-index.html

    STATE_JOINING  = "1"
    STATE_DONOR    = "2"
    STATE_DESYNCED = "2"
    STATE_DON_DSYN = "2"
    STATE_JOINED   = "3"
    STATE_SYNCED   = "4"

    // When looking for global varaibles with or without status

    STATUS     = true
    NOSTATUS   = false

)

var (
    gen_msg    string = "Percona XtraDB Cluster node is "

    ok_msg     string = gen_msg + "synced."
    err_nosync string = gen_msg + "not synced or non-PRIM."
    err_ro     string = gen_msg + "read-only."

    var_read_only           = "OFF"
)

// Get Global (status) variable from MySQL
//
// Parameters:
//
// - db: Pointer go sql.DB
// - name: What variable value to get
// - status: If is a global status

func getglobalvar(db *sql.DB, name string, status bool) (string, error) {

    var (
        variable_name  string
        value          string
        status_str     string = ""
    )

    if status {
        status_str = "STATUS"
    } else {
        status_str = "VARIABLES"
    }

    err := db.QueryRow("SHOW GLOBAL "+ status_str + " WHERE VARIABLE_NAME = '" + name + "';").Scan(&variable_name, &value)

    if err != nil {
        return err.Error(), err
    }

    return value, nil
}


// Get environment var, or fallback if environment var is empty
//
// Parameters:
//
// - ev_name: environnment var name
// - fallback: default value if not exist

func getenv(ev_name, fallback string) string {
    value := os.Getenv(ev_name)

    if len(value) == 0 {
        return fallback
    }

    return value
}

// Log message
//
// Parameters:
//
// - msg: String to write to log

func log(msg string) {

    t := time.Now()
    os.Stdout.Write([]byte( t.Format(time.RFC3339Nano) + "[CLUSTERCHECK] " + msg + "\n"))

}

// Check MySQL Percona cluster state
//
// Return:
//
// - nil: No errors
// - err: error content

func check_mysql() error {
    var (
        wsrep_local_state         string
        wsrep_cluster_status      string
    )

    // Get password from environment

    sql_user := "root"
    sql_pass := getenv("MYSQL_ROOT_PASSWORD", "")

    db_host  := getenv("MYSQL_CHECK_HOST","127.0.0.1:3306")

    available_when_donor    := getenv("AVAILABLE_WHEN_DONNOR", "") != ""
    //err_file                := getenv("ERR_FILE", "/dev/null")
    available_when_read_only:= getenv("AVAILABLE_WHEN_READONLY", "-1") == "0"

    attempt    := 1
    maxretries := 10

    for {
        conn, err := net.DialTimeout("tcp", db_host, 2 * time.Second )
        if err == nil {
            log("Connection "+ db_host + " Online")
            conn.Close()
            break
        }
        attempt++

        if attempt > maxretries {
            log("Connection retry limit reached")
            return err
        }

        time.Sleep(5 * time.Second)
        log("Connection sleep 5s")
    }

    db, err := sql.Open("mysql", sql_user + ":" + sql_pass + "@tcp(" + db_host + ")/mysql")

    if err != nil {
        log("sql.Open")
        return err
    }
    defer db.Close()

    // Test database connection

    attempt    = 1
    maxretries = 10

    for {
        err = db.Ping()
        if err == nil {
            break
        }
        attempt++

        if attempt > maxretries {
            // [CLUSTERCHECK] dial tcp 127.0.0.1:3306: getsockopt: connection refused
            log("db.ping retry limit reached")
            return err
        }

        log("db.ping sleep 1s")
        time.Sleep(1000 * time.Millisecond)
    }

    wsrep_local_state, err = getglobalvar(db, "wsrep_local_state", STATUS)

    if err != nil {
        log("get wsrep_local_state")
        return err
    }

    wsrep_cluster_status, err = getglobalvar(db, "wsrep_cluster_status", STATUS)

    if err != nil {
        log("get wsrep_cluster_status")
        return err
    }

    if (wsrep_cluster_status == "Primary" && wsrep_local_state == STATE_SYNCED) ||
                    (available_when_donor && wsrep_local_state == STATE_DON_DSYN) {

        if !available_when_read_only { // If not available when read only check if MySQL is on readonly
            var_read_only, err = getglobalvar(db, "read_only", NOSTATUS)
            if err != nil {
                log("get read_only")
                return err
            }
            if var_read_only == "ON" {
                log(err_ro)
                return errors.New(err_ro)
            }
        } // read_only = "OFF"

        // Percona XtraDB Cluster node local state is 'Synced' => return HTTP 200

        return nil
    }
    log("Not ready")
    return errors.New("Not ready")
}

// Send code 503 http service unavaliable and write msg
//
// Parameters:
//
// - w: Http response writer
// - r: http request
// - msg: Message to show on body content

func returnCode503msg(w http.ResponseWriter, r *http.Request, msg string) {
    // see http://golang.org/pkg/net/http/#pkg-constants
    w.WriteHeader(http.StatusServiceUnavailable)
    w.Write([]byte(msg + "\r\n"))
}

// Send code 200 http OK and write msg
//
// Parameters:
//
// - w: Http response writer
// - r: http request
// - msg: Message to show on body content

func returnCode200msg(w http.ResponseWriter, r *http.Request, msg string) {
    // see http://golang.org/pkg/net/http/#pkg-constants
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(msg + "\n\r"))
}

// Main http handler

func clustercheck(w http.ResponseWriter, r *http.Request) {
    if err := check_mysql(); err != nil {
        log("Err 503: " + err.Error())
        http.Error(w, err.Error(), 503)
    } else {
        log(ok_msg)
        returnCode200msg(w, r, ok_msg)
    }
}

// Main program entrypoint

func main() {

    args := os.Args[1:]
    if len(args) > 0 && args[0] == "-h" {
        help:= "Usage: \n\n" +
        "Environment variables:" + "\n\n" +
        "MYSQL_ROOT_PASSWORD: password to connect to MySQL" + "\n" +
        "MYSQL_CHECK_HOST: Host to connect, defaults to 127.0.0.1:3306" + "\n" +
        "AVAILABLE_WHEN_DONNOR: defaults to no (empty)" + "\n" +
        "AVAILABLE_WHEN_READONLY: defaults to -1 (not available)" + "\n"

        os.Stdout.Write([]byte(help))

    } else {

        mux := http.NewServeMux()
        mux.HandleFunc("/", clustercheck)

        // Port to listen

        http.ListenAndServe(":9200", mux)
    }
}
