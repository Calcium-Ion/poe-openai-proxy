module github.com/juzeon/poe-openai-proxy

go 1.19

require (
	github.com/Calcium-Ion/poe-api-go v1.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-resty/resty/v2 v2.7.0
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pelletier/go-toml/v2 v2.0.8
	github.com/pkoukk/tiktoken-go v0.1.4
	github.com/robfig/cron/v3 v3.0.1
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/bogdanfinn/fhttp v0.5.23 // indirect
	github.com/bogdanfinn/tls-client v1.4.0 // indirect
	github.com/bogdanfinn/utls v1.5.16 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/dop251/goja v0.0.0-20230707174833-636fdf960de1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/zhangyunhao116/fastrand v0.3.0 // indirect
	github.com/zhangyunhao116/skipmap v0.10.1 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// replace poe-api to local /usr/local/go/src/github.com/Calcium-Ion/poe-api
replace github.com/Calcium-Ion/poe-api-go => /Users/seefsl/GolandProjects/poe-api-go
