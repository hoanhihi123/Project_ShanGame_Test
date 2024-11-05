// Copyright 2020 The Nakama Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative chinese_poker_game_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative color_game.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative lobby_common_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative common_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative match_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative auth_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative wallet.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative bandarqq_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative slots_game.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative gaple_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative dragontiger_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative sicbo_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative baccarat_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative blackjack_api.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative operation.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative shangame_api.proto
