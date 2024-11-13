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

// giả sử userId = user quay lại game và muốn join lại trận đấu
// s  = match và trong này chứa thông tin của các user đã leave khỏi trận đấu

func (m *Engine) RejoinUserMessage(s *entity.MatchState, userId string) map[pb.OpCodeUpdate]proto.Message {
	// làm sao để lấy ra danh sách trận đấu ? làm sao để kiểm tra sự tồn tại của trận đấu ?

	// giả sử trận đấu tồn tại

	// check gamestate cho trường hợp dưới
	// trường hợp: trận đấu đang diễn ra
	// trường hợp: trận đấu đã kết thúc và đang khởi tạo ván chơi mới
	gameState := s.GetGameState()
	if gameState == pb.GameState_GameStatePlay { // state = play => chắc chắn trận đấu còn hợp lệ để xảy ra
		// add user lại vào trận đấu => add thông tin gì ? ,
		// từ userId => lấy thông tin của presenceLeave => add vào playingPresence
		leavePresence := s.GetInfoLeavePreseceByUserId(userId)
		s.PlayingPresences.Put(userId, leavePresence)
		s.PlayingPresences.Remove(userId) // xóa user khỏi presenceLeave
		// làm sao để nó tiếp tục chơi game được ?
		// còn phụ thuộc vào hệ thống xử lý như thế nào ?
	}
	// các trường hợp còn lại thì sao ? khác gì z ?

	// giả sử nhảy vào trường hợp còn lại => trận đấu ko tồn tại ?
	// => xếp user này vào 1 match bất kỳ còn đủ điều kiện để chơi.... bằng cách nào ?

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
