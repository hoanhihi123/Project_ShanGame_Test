package entity

import (
	"log"
	"strconv"

	pb "github.com/nakamaFramework/cgp-common/proto"

	"github.com/heroiclabs/nakama-common/runtime"
)

type ArrPbPlayer []*pb.Player

func NewPlayer(presence runtime.Presence) *pb.Player {
	p := pb.Player{
		Id:       presence.GetUserId(),
		UserName: presence.GetUsername(),
	}
	m, ok := presence.(MyPrecense)
	if ok {
		p.AvatarId = m.AvatarId
		p.VipLevel = m.VipLevel
		p.Wallet = strconv.FormatInt(m.Chips, 10)
	}
	return &p
}

func NewListPlayer(presences []runtime.Presence) ArrPbPlayer {
	listPlayer := make([]*pb.Player, 0, len(presences))
	for _, presence := range presences {
		p := NewPlayer(presence)
		listPlayer = append(listPlayer, p)
	}
	return listPlayer
}

func ConvertWalletFromStrToInteger(player pb.Player) int {
	chipPlayer, err := strconv.Atoi(player.Wallet)
	if err != nil {
		log.Println("can't convert int to string!")
	}
	return chipPlayer
}
