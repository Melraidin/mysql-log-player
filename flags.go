package main

import (
	"flag"
	"os"

	logger "github.com/500px/go-utils/chatty_logger"
	"github.com/500px/go-utils/flags"
	"fmt"
)

var (
	dbHost        = flag.String("db-host", "127.0.0.1", "MySQL host")
	dbName        = flag.String("db-name", "500px", "MySQL database name")
	dbUser        = flag.String("db-user", os.Getenv("DB_USER"), "MySQL username (env DB_USER)")
	dbPass        = flag.String("db-pass", "", "MySQL password (env DB_PASS)")
	dbConcurrency = flag.Int("db-concurrency", 1500, "Number of concurrent queries.")
	logLevel      = logger.LogSeverity("log", logger.INFO, "The log level to emit.")
	printVersion  = flag.Bool("version", false, "Print version information and exit.")

	sourcePath    = flag.String("source", "vc-data/20160317-13.gz", "Path of input log .gz or empty for stdin.")
	replaySpeed   = flag.Float64("replay-speed", 2.0, "replay speed 2.0 means twice as fast (half the time)")
	workerCount   = flag.Int("workers", 5, "Number of query workers.")
)

func parseFlags() {
	flags.SetFromEnv("db-user", "DB_USER")
	flags.SetFromEnv("db-pass", "DB_PASS")

	flag.Parse()
	logger.Init(*logLevel)

	if *printVersion {
		version()
		os.Exit(0)
	}

	if *dbUser == "" || *dbPass == "" {
		fmt.Println("Both db-user and db-pass (env DB_USER and DB_PASS) are required; for no password use \"''\"")
		os.Exit(1)
	}
	if *dbPass == "''" {
		*dbPass = ""
	}
}
