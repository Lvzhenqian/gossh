package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Lvzhenqian/sshtool"
	"golang.org/x/crypto/ssh"
	"gossh/conf"
	"io"
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
	c string
	a string
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
	flag.StringVar(&c, "c", "", "执行命令，以Index:CMD 来对某台机器执行ssh命令")
	flag.StringVar(&a, "a", "", "所有服务器执行命令CMD")
	flag.Usage = Usage
}

func Usage() {
	fmt.Fprintf(os.Stderr, `使用说明：
  gossh run@index command  对index的主机执行某条命令
  gossh user@index:path path  从远程拉取到本地
  gossh path user@index:path  从本地推送到远程
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
	title := &survey.Input{
		Message: "名称：",
	}
	survey.AskOne(title, &c.Name)

	address := &survey.Input{
		Message: "ip地址：",
	}
	survey.AskOne(address, &c.Data.IP)

	for {
		var p string
		port := &survey.Input{
			Message: "端口号：",
			Default: "22",
		}
		survey.AskOne(port, &p)
		if v, e := strconv.Atoi(p); e == nil {
			c.Data.Port = v
			break
		} else {
			fmt.Printf("\r")
			fmt.Println("请输入数字！！")
		}
	}

	u := &survey.Input{
		Message: "用户名：",
		Default: "root",
	}
	survey.AskOne(u, &c.Data.Username)

	var p string
	prompt := &survey.Password{
		Message: "密码：",
	}
	survey.AskOne(prompt, &p)
	if p == "" {
		c.Data.Password = ""
	} else {
		c.Data.Password = EncodePassword(p)
	}
	pk := &survey.Input{
		Message: "私钥路径：",
	}
	survey.AskOne(pk, &c.Data.PrivateKey)

	details := &survey.Input{
		Message: "备注：",
	}
	survey.AskOne(details, &c.Detail)

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
	//index := -1
	var result string
	var err error
	// 转为字符串slice
	var list []string
	list = append(list, "新增")
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
		for _, v := range cfg {
			list = append(list, v.Name)
		}
	}
	for {
		prompt := &survey.Select{
			Message: "选择你要登陆的主机：",
			Options: list,
			Default: "新增",
		}
		AskErr := survey.AskOne(prompt, &result)
		if AskErr == terminal.InterruptErr {
			os.Exit(0)
		} else if AskErr != nil {
			log.Fatal(err)
			return conf.SshConfig{}
		}

		if result == "新增" {
			n, err := AddOneConfig()
			if err == nil {
				list = append(list, n)
				if err := conf.Write(&cfg, ConfigPath); err != nil {
					log.Fatal("回写配置文件失败！")
				}
			}
		} else {
			break
		}
	}

	// 使用promptui 来显示菜单
	//for index < 0 {
	//	prompt := promptui.SelectWithAdd{
	//		Label:    "选择你要登陆的主机：",
	//		Items:    list,
	//		AddLabel: "新增",
	//	}
	//
	//	index, result, err = prompt.Run()
	//
	//	if index == -1 {
	//		n, err := AddOneConfig()
	//		if err == nil {
	//			list = append(list, n)
	//			if err := conf.Write(&cfg, ConfigPath); err != nil {
	//				log.Fatal("回写配置文件失败！")
	//			}
	//		}
	//	}
	//}

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

func ShowIdAndName() {
	for i, v := range cfg {
		fmt.Printf("index：%d name: %s \n", i, v.Name)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	switch {
	case h:
		flag.Usage()
	case s:
		ShowIdAndName()
	case a != "":
		for i := 0; i < len(cfg); i++ {
			cli := NewClient(cfg[i].Data)
			defer cli.Close()
			Ssh.Run(a, os.Stdout, cli)
		}
	case c != "":
		if !strings.Contains(c, ":") {
			panic("Error,not address or cmd in strings")
		}
		s := strings.Split(c, ":")
		index, _ := strconv.Atoi(s[0])
		cli := NewClient(cfg[index].Data)
		defer cli.Close()
		err := Ssh.Run(s[1], os.Stdout, cli)
		if err != nil {
			panic(err)
		}
	case flag.NArg() == 2:
		// username@address:path
		var (
			src, dst = args[0], args[1]
		)
		switch {
		case strings.Contains(src, ":"):
			if strings.Contains(src, "@") {
				panic("not found!!")
			} else {
				s := strings.Split(src, ":")
				index, err := strconv.Atoi(s[0])
				if err != nil {
					panic("请输入对应的配置文件ID!!")
				}
				cli := NewClient(cfg[index].Data)
				defer cli.Close()
				SshErr := Ssh.Get(s[1], dst, cli)
				if SshErr != nil {
					panic(err)
				}
			}
		case strings.Contains(dst, ":"):
			if strings.Contains(dst, "@") {
				panic("not found!!")
			} else {
				s := strings.Split(dst, ":")
				index, err := strconv.Atoi(s[0])
				if err != nil {
					panic("请输入对应的配置文件ID!!")
				}
				cli := NewClient(cfg[index].Data)
				defer cli.Close()
				SshErr := Ssh.Push(src, s[1], cli)
				if SshErr != nil {
					panic(err)
				}
			}
		default:
			s, _ := os.Open(src)
			defer s.Close()
			d, _ := os.Create(dst)
			defer d.Close()
			_, e := io.Copy(d, s)
			if e != nil {
				panic(e)
			}
		}
	default:
		conf := ShowList()
		cli := NewClient(conf)
		defer cli.Close()
		Ssh.Login(cli)
	}
}
