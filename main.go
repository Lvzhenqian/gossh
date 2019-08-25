package main

import (
	"encoding/base64"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/mewbak/gopass"
	"gossh/conf"
	"gossh/sshtool"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

const DefaultConfig = ".gossh.yaml"

var (
	ConfigPath string
	KEY        []byte
	cfg        conf.Config
)

func init() {
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
	if p, e := gopass.GetPass("密码："); e != nil {
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
	list = append(list,"exit")
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
	if result == "exit"{
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

func NewLogin(c conf.SshConfig) {
	var Ssh sshtool.SshClient
	var password string
	if c.Password != "" {
		password = DecodePassword(c.Password)
	}
	terminal := new(sshtool.SSHTerminal)
	Ssh = terminal
	client, err := sshtool.NewClient(c.IP, c.Port, c.Username, password, c.PrivateKey)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	e := Ssh.Login(client)
	if e != nil {
		fmt.Println("main error: ", e)
	}
}

func main() {
	cli := ShowList()
	NewLogin(cli)
}
