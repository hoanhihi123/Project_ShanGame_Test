package engine

import (
	"shangame-module/entity"

	pb "github.com/nakamaFramework/cgp-common/proto"

	"google.golang.org/protobuf/proto"
)

type Engine struct {
	deck *entity.Deck
}

func NewGameEngine() UseCase {
	return &Engine{}
}

func (m *Engine) NewGame(s *entity.MatchState) error {
	m.deck = entity.NewDeck()
	m.deck.Shuffle()
	s.Init()
	return nil
}

// số lượng thẻ bài muốn rút
// trả về các lá bài rút tương ứng
func (m *Engine) Deal(amount int) []*pb.Card {
	if list, err := m.deck.Deal(amount); err != nil {
		return nil
	} else {
		return list.Cards
	}
}

func (m *Engine) RejoinUserMessage(s *entity.MatchState, userId string) map[pb.OpCodeUpdate]proto.Message {
	return nil
}

func (m *Engine) Finish(s *entity.MatchState) *pb.ShanGameUpdateFinish {
	return s.CalcGameFinish()
}

func (m *Engine) Draw(s *entity.MatchState, userId string, handN0 pb.ShanGameHandN0) {
	s.AddCards(m.Deal(1), userId, handN0)
}

// func (m *Engine) DoubleDown(s *entity.MatchState, userId string, handN0 pb.ShanGameHandN0) int64 {
// 	s.AddCards(m.Deal(1), userId, handN0)
// 	return s.DoubleDownBet(userId, handN0)
// }
// func (m *Engine) Split(s *entity.MatchState, userId string) int64 {
// 	return s.SplitHand(userId)
// }
// func (m *Engine) Insurance(s *entity.MatchState, userId string) int64 {
// 	return s.InsuranceBet(userId)
// }
