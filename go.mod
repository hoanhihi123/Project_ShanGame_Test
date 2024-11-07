module shangame-module

go 1.23.2

// replace github.com/nakamaFramework/cgp-common => ./cgp-common

require (
	github.com/emirpasic/gods v1.18.1
	github.com/heroiclabs/nakama-common v1.34.0
)

require (
	github.com/stretchr/testify v1.9.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
)

require (
	github.com/bwmarrin/snowflake v0.3.0
	github.com/nakamaFramework/cgp-common v0.0.0-20241107020808-127a5c4028dc
	github.com/qmuntal/stateless v1.6.8
	go.uber.org/zap v1.24.0
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
)
