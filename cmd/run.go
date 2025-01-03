package cmd

import (
	"flag"
	"fmt"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/juju/errors"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"cardappcanal/global"
	"cardappcanal/metrics"
	"cardappcanal/router"
	"cardappcanal/service"
	"cardappcanal/storage"
	"cardappcanal/util/stringutil"
)

var (
	helpFlag     bool
	cfgPath      string
	stockFlag    bool
	positionFlag bool
	statusFlag   bool
)

func init() {
	flag.BoolVar(&helpFlag, "help", false, "this help")
	flag.StringVar(&cfgPath, "config", "app.yml", "application config file")
	flag.BoolVar(&stockFlag, "stock", false, "stock data import")
	flag.BoolVar(&positionFlag, "position", false, "set dump position")
	flag.BoolVar(&statusFlag, "status", false, "display application status")
	flag.Usage = usage
}

func Exec() {
	flag.Parse()
	if helpFlag {
		flag.Usage()
		return
	}

	// 初始化global
	err := global.Initialize(cfgPath)
	if err != nil {
		println(errors.ErrorStack(err))
		return
	}

	if stockFlag {
		doStock()
		return
	}

	// 初始化Storage
	err = storage.Initialize()
	if err != nil {
		println(errors.ErrorStack(err))
		return
	}

	if statusFlag {
		doStatus()
		return
	}

	if positionFlag {
		doPosition()
		return
	}

	initialize, err := service.Initialize()
	if err != nil {
		println(errors.ErrorStack(err))
		return
	}
	if err := metrics.Initialize(); err != nil {
		println(errors.ErrorStack(err))
		return
	}
	service.StartUp() // start application
	// 创建 Gin 实例
	router := router.InitRouter(initialize)

	go func() {
		if err := router.Run(":8181"); err != nil {
			fmt.Println("Server stopped:", err)
		}
	}()
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sin := <-s
	log.Printf("application stoped，signal: %s \n", sin.String())

	service.Close()
	storage.Close()

	//select {}
}

func doStock() {
	stock := service.NewStockService()
	if err := stock.Run(); err != nil {
		println(errors.ErrorStack(err))
	}
	stock.Close()
}

func doStatus() {
	ps := storage.NewPositionStorage()
	pos, _ := ps.Get()
	fmt.Printf("The current dump position is : %s %d \n", pos.Name, pos.Pos)
}

func doPosition() {
	others := flag.Args()
	if len(others) != 2 {
		println("error: please input the binlog's File and Position")
		return
	}
	f := "mysql-bin.001026"
	p := others[1]

	matched, _ := regexp.MatchString(".+\\.\\d+$", f)
	if !matched {
		println("error: The parameter File must be like: mysql-bin.000001")
		return
	}

	pp, err := stringutil.ToUint32(p)
	if nil != err {
		println("error: The parameter Position must be number")
		return
	}
	ps := storage.NewPositionStorage()
	pos := mysql.Position{
		Name: f,
		Pos:  pp,
	}
	ps.Save(pos)
	fmt.Printf("The current dump position is : %s %d \n", f, pp)
}

func usage() {
	fmt.Fprintf(os.Stderr, `version: 1.0.0
Usage: transfer [-c filename] [-s stock]

Options:
`)
	flag.PrintDefaults()
}
