module gossh

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.0.7
	github.com/Lvzhenqian/sshtool v0.1.2
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.7 // indirect
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.4
)

replace gopkg.in/alecthomas/kingpin.v3-unstable v3.0.0-20180810215634-df19058c872c => gopkg.in/alecthomas/kingpin.v2 v2.2.6
