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

// rút bài
func (m *Engine) Deal(amount int) []*pb.Card {
	if list, err := m.deck.Deal(amount); err != nil {
		return nil
	} else {
		return list.Cards
	}
}

// giả sử userId = user quay lại game và muốn join lại trận đấu

func (m *Engine) RejoinUserMessage(s *entity.MatchState, userId string) map[pb.OpCodeUpdate]proto.Message {

	// trước khi dùng func , người sử dụng func đã phải kiểm tra sự tồn tại r
	gameState := s.GetGameState()
	if gameState == pb.GameState_GameStatePlay && s != nil { // state = play => chắc chắn trận đấu còn hợp lệ để xảy ra

		s.LeavePresences.Remove(userId) // xóa user
		// opCodeArr := []pb.OpCodeUpdate{pb.OpCodeUpdate_OPCODE_USER_IN_TABLE_INFO, pb.OpCodeUpdate_OPCODE_UPDATE_TABLE}
		// opCodeArr2 := make(map[pb.OpCodeUpdate]proto.Message, 0)

		// message := proto.Message{}
		// opCodeArr2[pb.OpCodeUpdate_OPCODE_UPDATE_USER_INFO] = proto.MessageName(proto.Message.ProtoReflect().Get())
		// return opco
	}
	// giả sử nhảy vào trường hợp còn lại => trận đấu ko còn tồn tại => ko player ko thể join vì két thúc trận đấu => match remove

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
