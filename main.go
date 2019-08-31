package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/mewbak/gopass"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"gossh/conf"
	"gossh/sshtool"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultConfig = ".gossh.yaml"

var (
	Ssh        sshtool.SshClient
	ConfigPath string
	KEY        []byte
	cfg        conf.Config

	s bool
	h bool
)

func init() {
	// 创建一个terminal 实例给到ssh接口
	terminal := new(sshtool.SSHTerminal)
	Ssh = terminal
	// 读取当前运行用户信息
	use, e := user.Current()
	if e != nil {
		log.Fatal(e)
	}
	// aes 加密key 由环境变量GOSSHKEY来控制，如果没配置环境变量则使用当前运行的用户名来做为key
	if s, e := os.LookupEnv("GOSSHKEY"); e {
		KEY = []byte(s)
	} else {
		KEY = []byte(use.Username)
	}
	if s, e := os.LookupEnv("GOSSH"); e {
		ConfigPath, _ = filepath.Abs(s)
	} else {
		ConfigPath = filepath.Join(use.HomeDir, DefaultConfig)
	}
	// 如果配置文件不存在，则先创建一个空的配置，然后由程序回写。
	if _, e := os.Stat(ConfigPath); e != nil {
		f, e := os.Create(ConfigPath)
		if e != nil {
			panic(e)
		}
		defer f.Close()
		cfg = make(conf.Config, 0)
	} else {
		cfg = conf.Read(ConfigPath)
	}

	flag.BoolVar(&h, "h", false, "显示这个帮助页面")
	flag.BoolVar(&s, "s", false, "查看当前有那些主机，并显示对应的Index ID")
	flag.Usage = Usage
}

func Usage() {
	fmt.Fprintf(os.Stderr, `使用说明：
  gossh run@index command  对index的主机执行某条命令
  gossh send@index src dst   推送src到index主机的dst位置
  gossh get@index src dst   从index主机src位置拉取到本地dst位置
其他参数:
`)

	flag.PrintDefaults()
}

func EncodePassword(s string) string {
	encrypted := conf.AesEncryptECB([]byte(s), KEY)
	return base64.StdEncoding.EncodeToString(encrypted)
}

func DecodePassword(s string) string {
	b, e := base64.StdEncoding.DecodeString(s)
	if e != nil {
		panic(e)
	}
	decrypted := conf.AesDecryptECB(b, KEY)
	return string(decrypted)
}

func AddOneConfig() (name string, e error) {
	var c conf.Node
	fmt.Printf("名称：")
	fmt.Scanln(&c.Name)
	fmt.Printf("ip地址：")
	fmt.Scanln(&c.Data.IP)
	for {
		fmt.Printf("端口号：")
		if _, e := fmt.Scanln(&c.Data.Port); e != nil {
			fmt.Printf("\r")
			fmt.Println("请输入数字！！")
		} else {
			break
		}
	}
	fmt.Printf("用户名：")
	fmt.Scanln(&c.Data.Username)
	if p, e := gopass.GetPass("密码："); e != nil || p == "" {
		c.Data.Password = ""
	} else {
		c.Data.Password = EncodePassword(p)
	}
	fmt.Printf("私钥路径：")
	if _, e := fmt.Scanln(&c.Data.PrivateKey); e != nil {
		c.Data.PrivateKey = ""
	}
	fmt.Printf("备注：")
	fmt.Scanln(&c.Detail)
	c.Id = len(cfg)
	cfg = append(cfg, c)
	err := conf.Write(&cfg, ConfigPath)
	if err != nil {
		log.Fatal("添加节点失败！！")
		return "", err
	}
	return c.Name, nil
}

func ShowList() conf.SshConfig {
	index := -1
	var result string
	var err error
	// 转为字符串slice
	var list []string
	list = append(list, "exit")
	if len(cfg) == 0 {
		fmt.Println("配置文件为空，请先填写一个！！")
		n, err := AddOneConfig()
		if err == nil {
			list = append(list, n)
			if err := conf.Write(&cfg, ConfigPath); err != nil {
				log.Fatal("回写配置文件失败！")
			}
		}
	} else {
		for i, v := range cfg {
			fmt.Printf("%T, %T", i, v.Name)
			list = append(list, v.Name)
		}
	}

	// 使用promptui 来显示菜单
	for index < 0 {
		prompt := promptui.SelectWithAdd{
			Label:    "选择你要登陆的主机：",
			Items:    list,
			AddLabel: "新增",
		}

		index, result, err = prompt.Run()

		if index == -1 {
			n, err := AddOneConfig()
			if err == nil {
				list = append(list, n)
				if err := conf.Write(&cfg, ConfigPath); err != nil {
					log.Fatal("回写配置文件失败！")
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
		return conf.SshConfig{}
	}
	if result == "exit" {
		os.Exit(0)
	}
	// 输出对应的sshConfig
	for _, v := range cfg {

		if v.Name == result {
			return v.Data
		}
	}
	return conf.SshConfig{}
}

func NewClient(c conf.SshConfig) *ssh.Client {
	var password string
	if c.Password != "" {
		password = DecodePassword(c.Password)
	}
	client, err := sshtool.NewClient(c.IP, c.Port, c.Username, password, c.PrivateKey)
	if err != nil {
		panic(err)
	}
	return client
}

func GetConfig(keyword string) (string, conf.SshConfig) {
	s := strings.Split(keyword, "@")
	index, err := strconv.Atoi(s[1])
	if err != nil {
		panic("请输入对应的配置文件ID!!")
	}
	return s[0], cfg[index].Data
}

func ShowIdAndName() {
	for i, v := range cfg {
		fmt.Printf("index：%d name: %s \n", i, v.Name)
	}
}

func Get(src, dst string, c *ssh.Client) error {
	sftpCli, err := sftp.NewClient(c)
	if err != nil {
		return err
	}
	defer sftpCli.Close()
	state, Serr := sftpCli.Stat(src)
	if Serr != nil {
		return Serr
	}
	if state.IsDir() {
		return Ssh.GetDir(src, dst, c)
	} else {
		Dstat, _ := os.Stat(dst)
		if Dstat.IsDir() {
			return Ssh.GetFile(src, filepath.Join(dst, filepath.Base(src)), c)
		} else {
			return Ssh.GetFile(src, dst, c)
		}
	}
}

func Push(src, dst string, c *ssh.Client) error {
	Spath,err := filepath.Abs(src)
	if err != nil {
		panic(err)
	}
	Sstate, Serr := os.Stat(Spath)
	if Serr != nil{
		panic(err)
	}
	if Sstate.IsDir() {
		return Ssh.PushDir(Spath, dst, c)
	} else {
		sftpCli, err := sftp.NewClient(c)
		if err != nil {
			return err
		}
		defer sftpCli.Close()
		Dstat, err := sftpCli.Stat(dst)
		if err != nil{
			panic(err)
		}
		if Dstat.IsDir() {
			return Ssh.PushFile(Spath, filepath.Join(dst, filepath.Base(Spath)), c)
		} else {
			return Ssh.PushFile(Spath, dst, c)
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	//fmt.Println(args)
	if h {
		flag.Usage()
		os.Exit(0)
	}
	if s {
		ShowIdAndName()
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		conf := ShowList()
		cli := NewClient(conf)
		defer cli.Close()
		Ssh.Login(cli)
	} else {
		key, c := GetConfig(args[0])
		cli := NewClient(c)
		defer cli.Close()
		switch strings.ToLower(key) {
		case "send":
			err := Push(args[1], args[2], cli)
			if err != nil {
				panic(err)
			}
		case "get":
			err := Get(args[1], args[2], cli)
			if err != nil {
				panic(err)
			}
		case "run":
			cmd := strings.Join(args[1:], " ")
			err := Ssh.Run(cmd, cli)
			if err != nil {
				log.Fatal(err)
			}
		default:
			fmt.Println("错误的关键字，请按下列例子来使用！！")
			Usage()
		}
	}
}
