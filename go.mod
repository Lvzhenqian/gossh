module gossh

go 1.13

require (
	github.com/Lvzhenqian/sshtool v0.1.2
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/manifoldco/promptui v0.3.2
	github.com/mattn/go-runewidth v0.0.7 // indirect
	github.com/mewbak/gopass v0.0.0-20160315111356-fa08fb4d03e3
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	gopkg.in/alecthomas/kingpin.v3-unstable v3.0.0-20180810215634-df19058c872c // indirect
	gopkg.in/yaml.v2 v2.2.4
)

replace gopkg.in/alecthomas/kingpin.v3-unstable v3.0.0-20180810215634-df19058c872c => gopkg.in/alecthomas/kingpin.v2 v2.2.6
