package entity

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	pb "github.com/nakamaFramework/cgp-common/proto"
)

const (
	MinPresences  = 1
	MaxPresences  = 7
	MinBetAllowed = 10000
	MaxBetAllowed = 200000
)

type MatchState struct {
	baseMatchState

	allowBet       bool
	allowAction    bool
	visited        map[string]bool // trường này dùng để làm gì
	userBets       map[string]*pb.ShanGamePlayerBet
	userHands      map[string]*Hand
	dealerHand     *Hand
	currentTurn    string
	currentHand    pb.ShanGameHandN0 // trường này dùng để làm gì
	gameState      pb.GameState
	updateFinish   *pb.ShanGameUpdateFinish
	POT            int64 // nơi chứa tiền cược user và tiền user nhận được sau thắng
	playerIsDealer string
	moneyOfServer  int64
	deck           *Deck // tôi muốn mỗi 1 bộ bài tương ứng với 1 trận đấu
}

// create new match
func NewMatchState(label *MatchLabel) MatchState {
	return MatchState{
		baseMatchState: baseMatchState{
			Label:               label, // xác định loại trận đấu
			MinPresences:        MinPresences,
			MaxPresences:        MaxPresences,
			Presences:           linkedhashmap.New(), // người chơi có mặt tại trận đấu
			PlayingPresences:    linkedhashmap.New(), // người chơi đang tham gia trận đấu
			LeavePresences:      linkedhashmap.New(), // người chơi đã rời khỏi trận đấu
			PresencesNoInteract: make(map[string]int, 0),
			balanceResult:       nil,
		},
		allowBet:    false,
		allowAction: false,

		userBets:       make(map[string]*pb.ShanGamePlayerBet, 0), // tiền cược của các player
		userHands:      make(map[string]*Hand, 0),                 // những lá bài được chia cho player
		dealerHand:     &Hand{},                                   // những lá bài được chia cho dealer
		currentTurn:    "",                                        // đại diện cho lượt chơi, turn bet, turn play or something ...
		currentHand:    pb.ShanGameHandN0_SHANGAME_HAND_1ST,       // bộ bài số mấy của player
		gameState:      pb.GameState_GameStateIdle,                // trạng thái trò chơi (start or end)
		updateFinish:   nil,                                       // update trạng thái trận đấu
		playerIsDealer: "",
		deck:           NewDeck(),
	}
}

// init info of user bets, user hands
func (s *MatchState) Init() {
	// xóa hết userBet
	for k := range s.userBets {
		delete(s.userBets, k)
	}
	// xóa userHands
	for k := range s.userHands {
		delete(s.userHands, k)
	}
	s.balanceResult = nil // dùng để làm gì
	// xóa dealerHand
	s.dealerHand = &Hand{
		first: make([]*pb.Card, 0),
	}
	s.currentTurn = ""
	s.updateFinish = nil
	s.currentHand = pb.ShanGameHandN0_SHANGAME_HAND_1ST
}

func (s *MatchState) InitVisited() {
	s.visited = make(map[string]bool, 0)
	for k := range s.userHands {
		s.visited[k] = false
	}
}

func (s *MatchState) IsAllVisited() bool {
	if s.visited == nil {
		return false
	} else {
		for _, v := range s.visited {
			if !v {
				return false
			}
		}
		return true
	}
}

func (s *MatchState) SetVisited(userId string) {
	s.visited[userId] = true
}

func (s *MatchState) SetCurrentHandN0(v pb.ShanGameHandN0) { s.currentHand = v }
func (s *MatchState) GetCurrentHandN0() pb.ShanGameHandN0  { return s.currentHand }

func (s *MatchState) SetCurrentTurn(v string) { s.currentTurn = v }
func (s *MatchState) GetCurrentTurn() string  { return s.currentTurn }

func (s *MatchState) GetGameState() pb.GameState  { return s.gameState }
func (s *MatchState) SetGameState(v pb.GameState) { s.gameState = v }

func (s *MatchState) GetPlayerHand(userId string) *pb.ShanGamePlayerHand {
	return s.userHands[userId].ToPb()
}
func (s *MatchState) GetPlayerPartOfHand(userId string, pos pb.ShanGameHandN0) *pb.ShanGameHand {
	return s.userHands[userId].ToPb().First
}

func (s *MatchState) GetDealerHand() *pb.ShanGamePlayerHand {
	return s.dealerHand.ToPb()
}

// chia và set thông tin bài cho user
func (s *MatchState) AddCards(cards []*pb.Card, userId string, handN0 pb.ShanGameHandN0) {
	if userId == s.playerIsDealer {
		s.dealerHand.AddCards(cards)
	} else {
		if _, found := s.userHands[userId]; !found {
			s.userHands[userId] = &Hand{
				userId: userId,
				first:  make([]*pb.Card, 0),
			}
		}
		s.userHands[userId].AddCards(cards)
	}
}

func (s *MatchState) SetAllowBet(v bool) { s.allowBet = v }
func (s *MatchState) IsAllowBet() bool   { return s.allowBet }

func (s *MatchState) SetAllowAction(v bool) { s.allowAction = v }
func (s *MatchState) IsAllowAction() bool   { return s.allowAction }

func (s *MatchState) SetUpdateFinish(v *pb.ShanGameUpdateFinish) { s.updateFinish = v }
func (s *MatchState) GetUpdateFinish() *pb.ShanGameUpdateFinish  { return s.updateFinish }

func (s *MatchState) GetUserBetById(userId string) *pb.ShanGamePlayerBet { return s.userBets[userId] }

// check can bet ?
func (s *MatchState) IsCanBet(userId string, balance int64, bet *pb.ShanGameBet) bool {
	if bet.Chips+s.userBets[userId].First > int64(MinBetAllowed*10) {
		return true
	}
	if balance < bet.Chips {
		return false
	}
	return true
}

// check lại điều kiện, so sánh mức cược với tiền mà player có
// mức cược của user < min bet => true => user ko được tham gia match
func (m *MatchState) IsBetFitMatch(userId string) bool {
	return m.userBets[userId].First < int64(MinBetAllowed)
}

// set bet for user by id
func (s *MatchState) PlayerNotExits_inUserBets(userId string) {
	if _, found := s.userBets[userId]; !found { // không tìm thấy userId trong userBet thì set value các trường = 0
		s.userBets[userId] = &pb.ShanGamePlayerBet{
			UserId: userId,
			First:  0,
		}
	} else {
		log.Println("\nPlayer đã đặt cược trước đó = ", s.userBets[userId].First)
	}
}

func (s *MatchState) AddBet_inUserBets(userId string, chip int64) {
	if _, found := s.userBets[userId]; !found {
		s.userBets[userId] = &pb.ShanGamePlayerBet{
			UserId: userId,
			First:  0,
		}
	}
	s.userBets[userId].First += chip
	s.allowAction = false

}

func (s *MatchState) SubstractBet_inUserBets(userId string, chip int64) {
	fmt.Println("Số chip hiện tại :", s.userBets[userId].First)
	fmt.Println("Số chip muốn giảm : ", chip)
	// add bet for user by id
	s.userBets[userId].First -= chip
	fmt.Println("Số chip trong bet sau khi giảm: ", s.userBets[userId].First)

}

func (s *MatchState) SubstractMoney_inWallet_playingPresence(userId string, chip int64) {
	playingPresence := s.GetInfoPlayingPreseceByUserId(userId)
	playingPresence.Chips -= chip // trừ đi tiền trong ví tạm của player tương ứng
}

func (s *MatchState) AddMoney_inWallet_playingPresence(userId string, chip int64) {
	playingPresence := s.GetInfoPlayingPreseceByUserId(userId)
	playingPresence.Chips += chip // cộng tiền lại vào ví tạm thời của player
}

func (s *MatchState) AddMoney_inWallet_dealer(userId string, chip int64) {
	if s.playerIsDealer != "" {
		playingPresence := s.GetInfoPlayingPreseceByUserId(userId)
		playingPresence.Chips += chip // cộng tiền lại vào ví tạm thời của player
	}
}

// substract bet for user
func (s *MatchState) SubstractBetOfUserBet(v *pb.ShanGameBet) {
	// xem userId ko tồn tại trong userBet thì khởi tạo giá trị mới cho nó
	s.PlayerNotExits_inUserBets(v.UserId)

	// check money substract > userBet truoc do ko ?
	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

	s.SubstractBet_inUserBets(v.UserId, v.Chips)
	s.AddMoney_inWallet_playingPresence(v.UserId, v.Chips)

	fmt.Println("UserID current sau khi giảm mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)
}

func (s *MatchState) GetBet(v *pb.ShanGameBet) int64 {
	return s.userBets[v.UserId].First
}

// func (s *MatchState) IsCanSplitHand(userId string, balance int64) bool {
// 	if balance >= s.userBets[userId].First {
// 		return s.userHands[userId].PlayerCanSplit()
// 	}
// 	return false
// }

// func (s *MatchState) SplitHand(userId string) int64 {
// 	s.userBets[userId].Second = s.userBets[userId].First
// 	s.userHands[userId].Split()
// 	return s.userBets[userId].Second
// }

func (s *MatchState) IsCanHit(userId string, pos pb.ShanGameHandN0) bool {
	return s.userHands[userId].PlayerCanDraw(pos)
}

func (s *MatchState) IsBet(userId string) bool {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First > 0 {
		return true
	}
	return false
}

func (s *MatchState) CalcGameFinish() *pb.ShanGameUpdateFinish {
	result := &pb.ShanGameUpdateFinish{
		BetResults: make([]*pb.ShanGamePLayerBetResult, 0),
	}
	for _, h := range s.userHands {
		result.BetResults = append(result.BetResults, s.getPlayerBetResult(h.userId))
	}
	return result
}

func (s *MatchState) getResultEndGame() {
	fmt.Println("\n\n========= End Game : Result =========== ")

	for _, userHand := range s.userHands {
		result := int(0)
		if s.dealerHand != nil {
			result = userHand.Compare(s.dealerHand) // result after compare each player - dealer

			moneyWallet := s.GetMoneyIn_PresencesOfPlayer(userHand.userId)
			moneyPlay := s.GetMoneyOfPlayingPrecense(userHand.userId)
			betOfUser := s.userBets[userHand.userId].First
			ketQua := ""
			if result == 1 {
				ketQua = "Player win"
			} else if result == -1 {
				ketQua = "Player lose"
			} else {
				ketQua = "Hòa nhau"
			}

			fmt.Println("UserID = ", userHand.userId, ", Result: ", ketQua, " , mức cược = ", betOfUser, " money play = ", moneyPlay, " money wallet = ", moneyWallet)

		}
	}
	// xóa đi các user ko còn hợp lệ ra khỏi trận đấu
	s.removePlayerCantPlayContinue()

}

func (s *MatchState) removePlayerCantPlayContinue() {
	// list player ko còn đủ mức cược tối thiểu để chơi game
	// và set tiền trong playing về presence
	arr_userID_deletePlayingPrecence := s.getListPlayerNotConditionPlayContinue()

	// chuyển thông tin của player tại playing về presence
	// xóa thông tin player tại playingPresence
	s.removePlayerNotConditionPlayerContinue(arr_userID_deletePlayingPrecence)

	// xóa các player out game trong khi chơi ra khỏi match
	if len(s.LeavePresences.Values()) > 0 {
		for _, userIdLeave := range s.LeavePresences.Keys() {
			userId := userIdLeave.(string)
			// xóa userHand, userBet, playingPresence
			delete(s.userHands, userId)
			s.setBetToPresence(userId)
			delete(s.userBets, userId)
			s.PlayingPresences.Remove(userId)
			// giả sử đã set money of player vào chỗ khác rồi, xóa presences
			s.Presences.Remove(userId)
		}
	}
	//  presence set thông tin về cho player kiểu gì ? để xóa player ra khỏi hệ thống hoàn toàn chứ
	// kiểm tra dealerhand còn phù hợp điều kiện làm dealer hay ko ?
	s.checkDealerCurrentCanPlayContinue()

}

func (s *MatchState) checkDealerCurrentCanPlayContinue() {
	if s.playerIsDealer != "" {
		// check dealer còn đủ điều kiện hay ko
		if s.POT < int64(MinBetAllowed) {
			s.SetUserChipsInWallet(s.playerIsDealer, s.POT) // set tiền trong ví user hiện tại = chip còn lại
			s.PlayingPresences.Remove(s.dealerHand.userId)  // remove player khỏi playingPresence
			s.SetBetForServerIsDealer()                     // set lại quyền làm dealer cho server
		}

	}
}

func (s *MatchState) removePlayerNotConditionPlayerContinue(arr_userID_deletePlayingPrecence []string) {
	for _, userID_delete := range arr_userID_deletePlayingPrecence {
		fmt.Println("user Id delete = ", userID_delete)
		// xóa userHand, userBet, playingPresence
		delete(s.userHands, userID_delete)
		// s.setBetToPresence(userID_delete)
		delete(s.userBets, userID_delete)
		s.PlayingPresences.Remove(userID_delete)
		// giả sử đã set money of player vào chỗ khác rồi, xóa presences
		s.Presences.Remove(userID_delete)
	}

}

func (s *MatchState) getListPlayerNotConditionPlayContinue() []string {
	arr_userID_deletePlayingPrecence := []string{}

	for _, key := range s.PlayingPresences.Keys() {
		userId := key.(string)
		value, ok := s.Presences.Get(key)
		presence := value.(MyPrecense)
		chipPlayerCurrent := presence.Chips
		if ok {
			if chipPlayerCurrent < int64(MinBetAllowed) {
				s.SetUserChipsInWallet(userId, chipPlayerCurrent) // set tiền trong ví user hiện tại = chip còn lại
				arr_userID_deletePlayingPrecence = append(arr_userID_deletePlayingPrecence, userId)
			}
		}
	}
	return arr_userID_deletePlayingPrecence
}

func (s *MatchState) setBetToPresence(userId string) {
	if presence, exist := s.Presences.Get(userId); exist {
		presence := presence.(MyPrecense)

		presence.Chips += s.GetBetOfUser_byID(userId)
		s.Presences.Put(userId, presence)
	}

}

func (s *MatchState) GetMoneyIn_PresencesOfPlayer(userId string) int {
	if value, exists := s.Presences.Get(userId); exists {
		if exists {
			presence := value.(MyPrecense)
			return int(presence.Chips)
		}
	}
	return 0
}

func (s *MatchState) GetMoneyIn_PlayingPresencesOfPlayer(userId string) int {
	if value, exists := s.PlayingPresences.Get(userId); exists {
		if exists {
			playingPrecences := value.(MyPrecense)
			return int(playingPrecences.Chips)
		}
	}
	return 0
}

// lấy kết quả cược của người chơi
func (s *MatchState) getPlayerBetResult(userId string) *pb.ShanGamePLayerBetResult {

	userBet := s.userBets[userId]
	r1 := s.userHands[userId].Compare(s.dealerHand)
	tiLeThang_player := s.userHands[userId].GetTiLeThangThuaPlayer()
	tiLeThang_dealer := s.dealerHand.GetTiLeThangThuaPlayer()
	phanTram_tienHo_player := s.GetTiLeTienHo_User(userId)
	phanTram_tienHo_dealer := s.GetTiLeTienHo_User(s.dealerHand.userId)

	first := &pb.ShanGameBetResult{
		BetAmount: userBet.First,
		WinAmount: 0,
		Total:     userBet.First,
	}
	// tổng tiền thắng = winAmount*tỉ lệ thắng
	// % tiền hồ = 5
	// với số tiền thắng - tiền hồ

	if first.BetAmount > 0 {
		first.IsWin = int32(r1)
		if r1 > 0 { // player win
			first.WinAmount = first.BetAmount
			first.Total = (phanTram_tienHo_player / 100) * (first.WinAmount * tiLeThang_player)
		} else if r1 < 0 { // player lose
			first.WinAmount = -first.BetAmount
			first.Total = (phanTram_tienHo_dealer / 100) * (first.WinAmount * tiLeThang_dealer)
		}
	}

	return &pb.ShanGamePLayerBetResult{
		UserId: userId,
		First:  first,
	}
}

func (s *MatchState) GetLegalActions() []pb.ShanGameActionCode {
	result := make([]pb.ShanGameActionCode, 0)
	if s.userHands[s.currentTurn].PlayerCanDraw(s.currentHand) {
		result = append(result, pb.ShanGameActionCode_SHANGAME_ACTION_HIT)

		result = append(result, pb.ShanGameActionCode_SHANGAME_ACTION_STAY)
	}
	return result
}

func (s *MatchState) DealerPotentialBlackjack() bool {
	return s.dealerHand.DealerPotentialBlackjack()
}

func (s *MatchState) IsDealerMustDraw() bool {
	return s.dealerHand.DealerMustDraw()
}

func (s *MatchState) IsGameEnded() bool {
	return s.updateFinish != nil
}

func (s *MatchState) GetPlayerChipsInWallet(userID string) int64 {
	if value, ok := s.PlayingPresences.Get(userID); ok {
		playing_presence := value.(MyPrecense)
		return playing_presence.Chips
	}
	return 0
}

// set money of player in wallet
func (s *MatchState) SetUserChipsInWallet(userID string, moneyChange int64) {
	if value, ok := s.PlayingPresences.Get(userID); ok {
		playing_presence := value.(MyPrecense) // Ép kiểu
		playing_presence.Chips = moneyChange   // Cập nhật giá trị Chips của player lose
		s.Presences.Put(userID, playing_presence)
		fmt.Println("Tiền của ", userID, "sau ván chơi tại playingPresen có giá trị = ", playing_presence.Chips)
	}
}

func (s *MatchState) IdentifyWhoWin() ([]string, []string) {
	fmt.Println("\nXác định danh sách player win, danh sách dealer win")

	// ai thắng hay thua
	userId_playerWin := []string{}
	userId_dealerWin := []string{}

	// duyệt userhand => compare and get list userWin
	// win, lose, hòa
	// chỉ xác định ai win => trừ tiền, việc trừ tiền từ ai xử lý ntn nằm ở hàm tính tiền
	if len(s.userHands) > 0 {
		for _, userHand := range s.userHands {
			result := userHand.Compare(s.dealerHand)
			if result == 1 {
				userId_playerWin = append(userId_playerWin, userHand.userId)
			} else if result == -1 {
				userId_dealerWin = append(userId_dealerWin, userHand.userId)
			} else {
				continue
			}
		}
	}

	return userId_playerWin, userId_dealerWin
}

func (s *MatchState) CalMoney_whoWin() {
	fmt.Println("================================ Tính tiền ==================")
	fmt.Println("Số lượng người dùng: ", len(s.userHands))
	fmt.Printf("Thông tin của dealerHand current : %+v", s.dealerHand.first)

	userIdPlayerWin, userIdPlayerLose := s.IdentifyWhoWin()

	// tính tiền thắng cho player
	if s.playerIsDealer == "" && len(userIdPlayerWin) > 0 {
		s.CalMoneyForPlayerWin_dealerIsServer(userIdPlayerWin) // tính tiền cho các player win
	} else {
		fmt.Println(".... tính tiền thắng cho player .... với dealer is player")
		s.CalMoneyForPlayerWin_dealerIsPlayer(userIdPlayerWin) // tính tiền cho các player win
	}

	// tính tiền thắng cho dealer
	if len(userIdPlayerLose) > 0 {
		s.CalMoneyFor_dealerWin(userIdPlayerLose) // tính tiền cho dealer win
	}

}

func (s *MatchState) SetMoneyForDealerWin(
	userIdLose string, moneyDealerWin int64,
	moneyDealerWin_fact int64, moneySetPlayerLose int64) {
	if s.playerIsDealer != "" {
		s.AddMoneyForDealerWin(moneyDealerWin_fact)
	} else {
		s.AddMoneyForDealerWin(moneyDealerWin)
	}
	// + tiền dealer win
	s.SubstractMoneyForPlayerToPlayingPrecence(userIdLose, moneySetPlayerLose) // - tiền player lose

}

// chỉ khác nhau là cộng tiền vào đâu thôi
func (s *MatchState) CalMoneyFor_dealerWin(userIdLoses []string) {
	fmt.Println("\nTính tiền thắng cho dealer win - tại func calMoneyFor_dealerWin() ")
	for _, userIdLose := range userIdLoses {
		fmt.Println("User_", userIdLose, " LOSE dealer_", s.playerIsDealer)
		moneyInWalletOfPlayer := int64(s.GetMoneyIn_PlayingPresencesOfPlayer(userIdLose))
		fmt.Println("Tiền trong ví còn lại của player = ", moneyInWalletOfPlayer)
		moneyBetOfPlayer := s.GetBetOfUser_byID(userIdLose)

		fmt.Println("Tiền đặt cược của player = ", moneyBetOfPlayer)
		tiLeThangDealer := s.dealerHand.GetTiLeThangThuaPlayer()
		fmt.Println("Tỉ lệ thắng dealer = ", tiLeThangDealer)
		fmt.Println("userId_dealerHand = ", s.dealerHand.userId)
		// tiền của dealer lưu tại đâu
		moneyDealerWin := s.GetBetOfUser_byID(s.dealerHand.userId) * int64(tiLeThangDealer)
		fmt.Println("Tiền dealer win = ", moneyDealerWin)

		phanTramTienHo := s.GetTiLeTienHo_User(s.dealerHand.userId)
		fmt.Println("Phần trăm tiền hồ dealer phải trả = ", phanTramTienHo)
		tienHo_playerPaid := int64((float64(phanTramTienHo) / 100) * float64(moneyDealerWin))
		fmt.Println("Tiền hồ mà player phải trả = ", tienHo_playerPaid)

		moneyDealerWin_fact := moneyDealerWin - tienHo_playerPaid
		fmt.Println("Tiền thắng nhận của player = ", moneyDealerWin_fact)

		if s.playerIsDealer != "" {
			s.moneyOfServer += tienHo_playerPaid
		}

		if moneyInWalletOfPlayer >= moneyDealerWin { // player đủ tiền trả
			s.SetMoneyForDealerWin(userIdLose, moneyDealerWin, moneyDealerWin_fact, moneyDealerWin)
		} else { // player ko đủ tiền trả
			s.SetMoneyForDealerWin(userIdLose, moneyDealerWin, moneyDealerWin_fact, moneyBetOfPlayer)
		}
	}
}

func (s *MatchState) CalMoneyForPlayerWin_dealerIsServer(userIdWin []string) {
	fmt.Println(".... tính tiền thắng cho player .... với dealer is server")
	if len(userIdWin) > 0 {
		for _, userIdWin := range userIdWin {
			fmt.Println("\nPlayer_", userIdWin, " WIN Dealer_", s.playerIsDealer)
			for _, userHand := range s.userHands {
				if userHand.userId == userIdWin {
					s.AddMoneyForPlayerToPlayingPrecence(userIdWin, s.GetMoneyWinOfPlayer(userIdWin, userHand))
				}
			}
		}

	}
}

func (s *MatchState) CheckDealerHaveEnoughMoneyPaid(userIdWin []string) int64 {
	// tiền mà dealer đang có
	sumMoneyDealer := s.POT
	sumMoney_playerWin := int64(0)
	for _, userIdWin := range userIdWin {
		for _, userHand := range s.userHands {
			if userHand.userId == userIdWin {
				// cộng tiền cho player_win vào playingPresence
				sumMoney_playerWin += s.GetMoneyOfPlayingPrecense(userIdWin)
			}
		}
	}
	return sumMoneyDealer - sumMoney_playerWin
}

func (s *MatchState) CalMoneyForPlayerWin_dealerIsPlayer(userIdWin []string) {
	if len(userIdWin) > 0 {
		sumPlayerWin := s.CheckDealerHaveEnoughMoneyPaid(userIdWin)
		fmt.Println("Tổng tiền các user win = ", sumPlayerWin)
		if sumPlayerWin >= 0 {
			for _, userIdWin := range userIdWin {
				fmt.Println("\nPlayer_", userIdWin, " WIN Dealer_", s.playerIsDealer)
				for _, userHand := range s.userHands {
					if userHand.userId == userIdWin {
						// update wallet of user
						s.AddMoneyForPlayerToPlayingPrecence(userIdWin, s.GetMoneyWinOfPlayer(userIdWin, userHand))
						// trừ tiền của dealer tại POT
						s.POT -= sumPlayerWin
					}
				}
			}
		} else {
			for _, userIdWin := range userIdWin {
				for _, userHand := range s.userHands {
					if userHand.userId == userIdWin {
						// update wallet of user
						s.AddMoneyForPlayerToPlayingPrecence(userIdWin, s.GetMoneyWinOfPlayer_DealerIsPlayerNoEnoughMoneyToPaid(userIdWin, userHand, sumPlayerWin))
					}
				}
			}
		}

	}
}

func (s *MatchState) GetMoneyWinOfPlayer(userIdWin string, userHand *Hand) int64 {
	fmt.Println("\nChạy vào function getMoneyWinOfPlayer() ")
	tiLeThang_player := userHand.GetTiLeThangThuaPlayer()
	moneyWin_player_fact := s.GetBetOfUser_byID(userIdWin) * int64(tiLeThang_player)
	phanTramTienHo := s.GetTiLeTienHo_User(userIdWin)
	tienHo_playerPaid := int64((float64(phanTramTienHo) / 100) * float64(moneyWin_player_fact))

	s.moneyOfServer += tienHo_playerPaid // cộng tiền hồ vào cho server
	moneyWin_player_after := moneyWin_player_fact - tienHo_playerPaid

	fmt.Println("\n\nTỉ lệ thắng player = ", tiLeThang_player)
	fmt.Println("Tiền mà player win chưa trừ VAT = ", moneyWin_player_fact)
	fmt.Println("Phần trăm tiền hồ của player = ", phanTramTienHo)
	fmt.Println("Tiền hồ player phải trả server = ", tienHo_playerPaid)
	fmt.Println("Tiền mà player thực nhận = ", moneyWin_player_after)
	return moneyWin_player_after
}

func (s *MatchState) GetMoneyWinOfPlayer_DealerIsPlayerNoEnoughMoneyToPaid(userIdWin string, userHand *Hand, sumPlayerWin int64) int64 {
	fmt.Println("Chạy vào func ")
	tiLeThang_player := int64(userHand.GetTiLeThangThuaPlayer())
	moneyUserWin := s.GetMoneyWinOfPlayer(userIdWin, userHand)
	moneyRemainingOfDealer := s.POT
	moneyPlayerReceive := ((moneyUserWin / sumPlayerWin) * moneyRemainingOfDealer) * tiLeThang_player
	phanTramTienHo := s.GetTiLeTienHo_User(userIdWin)
	tienHo_playerPaid := int64((float64(phanTramTienHo) / 100)) * moneyPlayerReceive
	s.moneyOfServer += tienHo_playerPaid // cộng tiền hồ vào cho server
	moneyWin_player_after := moneyPlayerReceive - tienHo_playerPaid

	fmt.Println("\n\nTỉ lệ thắng player = ", tiLeThang_player)
	fmt.Println("Tiền mà player win chưa trừ VAT = ", moneyUserWin)
	fmt.Println("Phần trăm tiền hồ của player = ", phanTramTienHo)
	fmt.Println("Tiền hồ player phải trả = ", tienHo_playerPaid)
	fmt.Println("Tiền mà player nhận = ", moneyPlayerReceive)
	fmt.Println("Tiền mà player thực nhận = ", moneyWin_player_after)
	return moneyWin_player_after
}

func (s *MatchState) DeleteUserHand_AfterCalMoney(userIdDelete []string) {
	for _, userId := range userIdDelete {
		delete(s.userHands, userId)
	}
}

func (s *MatchState) AddMoneyForPlayerToPlayingPrecence(userId string, chipUpdate int64) {
	// lấy ra playingPrecence hiện tại
	playingPresence, ok := s.PlayingPresences.Get(userId)

	if ok {
		presence := playingPresence.(MyPrecense)
		presence.Chips += chipUpdate
		s.PlayingPresences.Put(userId, presence)
		fmt.Printf("player _ %v sau khi cập nhật %+v", userId, presence)

	} else {
		log.Fatal("Không tìm thấy userID_", userId, " tại func addMoneyForPlayerToPlayingPrecence()!")
	}
}

func (s *MatchState) SubstractMoneyForPlayerToPlayingPrecence(userId string, chipUpdate int64) {
	// lấy ra playingPrecence hiện tại
	playingPresence, ok := s.PlayingPresences.Get(userId)

	if ok {
		presence := playingPresence.(MyPrecense)
		presence.Chips -= chipUpdate
		s.PlayingPresences.Put(userId, presence)
	} else {
		log.Fatal("Không tìm thấy userID_", userId, " tại func addMoneyForPlayerToPlayingPrecence()!")
	}
}

func (s *MatchState) AddMoneyForDealerWin(chipUpdate int64) {
	if s.playerIsDealer == "" {
		s.moneyOfServer += chipUpdate
	} else {
		s.POT += chipUpdate
	}

}

func (s *MatchState) GetMoneyOfPlayingPrecense(userId string) int64 {
	// lấy ra playingPrecence hiện tại
	playingPresence, ok := s.PlayingPresences.Get(userId)

	if ok {
		presence := playingPresence.(MyPrecense)
		return presence.Chips
	} else {
		log.Fatal("Không tìm thấy userID_", userId, " tại func addMoneyForPlayerToPlayingPrecence()!")
		return 0
	}
}

// get tỉ lệ tiền hồ của mỗi player win
func (s *MatchState) GetTiLeTienHo_User(userID string) int64 {
	vipLevel := int64(0)     // vipLevel of user
	bankOfPlayer := int64(0) // total money of player
	tiLeTienCuoc := int64(0)
	// làm sao để lấy ra vip của player
	if value, ok := s.Presences.Get(userID); ok {
		presence := value.(MyPrecense)                           // Ép kiểu về MyPrecense
		vipLevel = presence.VipLevel                             // Gán giá trị Chips cho biến mới
		bankOfPlayer = presence.Chips + s.userBets[userID].First // ? có tính cả mức cược mà presence đã cược hay ko ? // may be , có , vì nếu wallet ko đủ => cũng phải dùng bet
		tiLeTienCuoc = (s.userBets[userID].First / presence.Chips) / 100
	}

	// lấy ra tỉ lệ cược của mỗi user tương ứng theo vipLevel
	if vipLevel == 0 {
		if (bankOfPlayer >= 0 && bankOfPlayer <= 100000000000) && (tiLeTienCuoc >= 0 && tiLeTienCuoc <= 100) {
			return 5
		}
	} else if vipLevel == 1 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 8
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 85 {
				return 12
			} else if tiLeTienCuoc > 85 && tiLeTienCuoc <= 100 {
				return 15
			}
		} else if bankOfPlayer <= 100000000000 {
			// eg (0-100), eg (0-60), eg (60-85), eg (85-100)

			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 8
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc > 85 && tiLeTienCuoc <= 100 {
				return 20
			}
		}
	} else if vipLevel <= 4 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 2000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 8
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 12
			}
		} else if bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 10
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 20
			}
		}
	} else if vipLevel <= 10 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 5000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 8
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 10
			} else if tiLeTienCuoc <= 100 {
				return 12
			}
		} else if bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 10
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 20
			}
		}
	}

	return 0

}

func (s *MatchState) GetBetOfUser_byID(userId string) int64 {
	if bet, exist := s.userBets[userId]; exist {
		return bet.First
	}
	log.Println("UserId _", userId, " không tồn tại trong userBet!, Error at func GetBetOfUser_byID()!")
	return 0
}

// chia 2 lá bài cho mọi player chưa được chia bài
func (s *MatchState) ChiaBaiChoPlayerTuongUng() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets { // lấy danh sách userBet - lấy ra userID tham gia
			// fmt.Println("Chia bài cho user hiện tại có id: " + userBet.UserId)
			// fmt.Println("s.deck.Dealt trước đó = ", s.deck.Dealt)
			listCard_chia2La, err := s.deck.Deal(2) // mỗi userBet - lấy ra 2 lá bài trong bộ bài

			if err != nil {
				// log.Println("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
				s.deck = NewDeck()
				// fmt.Println("s.deck.Dealt = ", s.deck.Dealt)
			}

			// chia bài cho dealer
			if userBet.UserId == s.playerIsDealer {
				// trường hợp chưa được chia lá bài nào
				// fmt.Println("\nChia bài cho dealer, với id = " + userBet.UserId)
				if len(s.dealerHand.first) == 0 { // dealerhand chưa được khởi tạo và thêm giá trị nào
					s.SetInfoForDealerHands(userBet.UserId, listCard_chia2La.Cards)
				}
				// fmt.Printf("Bộ bài của dealer sau khi chia bài: %+v\n", s.dealerHand.first)
			} else { // chia bài cho player
				s.ChiaBaiChoUserHand(userBet.UserId, listCard_chia2La.Cards)
				// fmt.Printf("Bộ bài của player sau khi chia bài: %+v\n", s.userHands[userBet.UserId].first)
			}
		}
	} else {
		log.Fatal("Chưa có user nào đặt cược vào ván chơi!")
	}
}

func (s *MatchState) CheckExistUserHandById(userId string) bool {
	if _, exist := s.userHands[userId]; exist {
		return true
	}
	return false
}

func (s *MatchState) ChiaBaiChoUserHand(userId string, listCard []*pb.Card) {
	checkExistUserHand := s.CheckExistUserHandById(userId)
	if !checkExistUserHand {
		s.SetInfoForUserHands(userId, listCard)
	}
}

func (s *MatchState) SetInfoForUserHands(userId string, listCard []*pb.Card) {
	s.userHands[userId] = &Hand{
		userId: userId,
		first:  listCard,
	}
}

func (s *MatchState) SetInfoForDealerHands(userId string, listCard []*pb.Card) {
	s.dealerHand = &Hand{
		userId: userId,
		first:  listCard,
	}
}

func (s *MatchState) GetInfoPreseceByUserId(userId string) MyPrecense {
	presene, ok := s.Presences.Get(userId)
	if !ok {
		log.Println("Not have this presence!")
		return MyPrecense{}
	} else {
		return presene.(MyPrecense)
	}
}

func (s *MatchState) GetInfoPlayingPreseceByUserId(userId string) MyPrecense {
	presene, ok := s.PlayingPresences.Get(userId)
	if !ok {
		log.Println("Not have this Playing presence! with userId _", userId)
		return MyPrecense{}
	} else {
		return presene.(MyPrecense)
	}
}

func (s *MatchState) GetInfoLeavePreseceByUserId(userId string) MyPrecense {
	presene, ok := s.LeavePresences.Get(userId)
	if !ok {
		log.Println("Not have this Playing presence! with userId _", userId)
		return MyPrecense{}
	} else {
		return presene.(MyPrecense)
	}
}

func (s *MatchState) SetBetForServerIsDealer() {
	if s.playerIsDealer == "" {
		// kiểm tra userId là key của userBet đã có chưa => nếu chưa có thì tạo mới và trả về userID
		s.PlayerNotExits_inUserBets("")
		s.userBets[""].First = int64(s.Label.Bet)
	}
}

// pot is bank of dealer for every dealer
func (s *MatchState) SetInfoForDealer(userId string) {

	// pot, dealerhand, userBet
	if s.playerIsDealer == "" {
		// POT of server là vô hạn
		s.dealerHand = &Hand{
			userId: "",
		}
		s.SetBetForServerIsDealer()
	} else {
		presenceRegisteDealer := s.GetInfoPreseceByUserId(userId)
		s.POT = presenceRegisteDealer.Chips
		s.playerIsDealer = userId
		s.dealerHand = &Hand{
			userId: userId,
		}
		// set bet for player is dealer đồng thời với các player khác
	}

}

func (s *MatchState) Player_RegisterDealer(userId_param string) {
	presence := s.GetInfoPreseceByUserId(userId_param)

	// player is dealer
	if presence.Chips > int64(MinBetAllowed*10) { // so sánh với mức cược tối thiểu của trận đấu
		s.SetInfoForDealer(userId_param)
	} else { // server is dealer
		s.SetInfoForDealer("")
	}
}

func (s *MatchState) Set_PlayerCanBeDealer(idPlayerReplace string) {
	playerPresence_1 := s.GetInfoPreseceByUserId(s.playerIsDealer)
	// lấy mức chip của player đang là dealer
	chip_playerIsDealerCurrent := playerPresence_1.Chips

	playerPresence_2 := s.GetInfoPreseceByUserId(idPlayerReplace)
	// lấy mức chip của player khác muốn xin làm dealer thay thế
	chip_player2 := playerPresence_2.Chips

	// check điều kiện đủ để player khác xin làm dealer - thay thế dealer current
	if chip_player2 > chip_playerIsDealerCurrent && chip_player2 > int64(MinBetAllowed) {
		s.SetInfoForDealer(idPlayerReplace)
	}
}

func (s *MatchState) AddPresence_ToPlayingPrecense_InMatch() {
	// add dữ liệu từ precense vào playingprecense
	if s.Presences.Size() > 0 {
		for _, userId := range s.Presences.Keys() {
			userId, ok := userId.(string)
			if !ok {
				log.Fatal("Error get userId at func addPresence_ToPlayingPrecense_InMatch()!")
			}
			presence := s.GetInfoPreseceByUserId(userId)
			if presence.Chips >= int64(MinBetAllowed)*10 {
				if s.PlayingPresences.Size() <= 7 {
					s.PlayingPresences.Put(userId, presence)
				}
			}

		}
	}
}

func (s *MatchState) SetAddBet_forPlayerAndDealer() {
	// duyệt danh sách playing precense => set mức đặt cược theo % đặt cược random
	fmt.Println("Set mức cược cho các player .... ")
	for _, key := range s.PlayingPresences.Keys() {
		userId, ok := key.(string)
		if !ok {
			log.Fatal("UserId _", userId, " không tồn tại, Error happen at func setAddBet_forPlayerAndDealer()! ")
		}
		player := s.GetInfoPreseceByUserId(userId)
		percentBet := int64(s.RandDomPercentBet())
		chipsUserBet := player.Chips * (100 - percentBet) / 100

		fmt.Println("Percent Bet : ", percentBet)
		fmt.Println("UserId = ", userId, ", Đặt cược = ", chipsUserBet, " tiền trong ví trước đó = ", player.Chips)
		// if userId này chưa tồn tại => khởi tạo 1 userBet rỗng
		s.PlayerNotExits_inUserBets(userId)

		s.AddBet_inUserBets(userId, chipsUserBet)

		s.SubstractMoney_inWallet_playingPresence(userId, chipsUserBet)

	}

	if s.playerIsDealer == "" {
		fmt.Println("Set mức cược cho dealer nếu dealer là server")
		s.SetBetForServerIsDealer()
	}

}

func (s *MatchState) SetSubstractBet_forPlayerAndDealer() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets {

			chipsUserBet := s.GetBetOfUser_byID(userBet.UserId) * (100 - int64(s.RandDomPercentBet())) / 100
			fmt.Println("UserId = ", userBet.UserId, ", Đặt cược = ", chipsUserBet, " tiền đặt cược trước đó = ", userBet.First)

			s.SubstractBet_inUserBets(userBet.UserId, chipsUserBet)
			if userBet.UserId == s.playerIsDealer && s.playerIsDealer == "" {
				// add mức cược cho dealer
				s.userBets[""].First += chipsUserBet
			} else {
				s.AddMoney_inWallet_playingPresence(userBet.UserId, chipsUserBet)
			}
		}
	} else {
		log.Println("Không có player nào đặt cược _ để thực hiện giảm mức cược")
	}
}

func (s *MatchState) CheckDealerHand_haveTypeShan() bool {
	dealerPoint, dealerHand := s.dealerHand.Eval() // tính điểm cho dealer và lấy ra type của nó
	return dealerPoint > 0 && dealerHand == pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN
}

func (s *MatchState) IsPlayerHave_TypeShan(userIdPlayer string) bool {
	playerPoint, playerTypeHand := s.userHands[userIdPlayer].Eval()
	return playerPoint >= 0 && playerTypeHand != pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN && len(s.userHands[userIdPlayer].first) <= 3
}

func (s *MatchState) GetRandomPlayerBocBai() []string {
	userIdBocBai := []string{}
	dem := int(0)
	for _, key := range s.PlayingPresences.Keys() {
		userId, ok := key.(string)
		if ok {
			if dem == 2 {
				break
			}
			userIdBocBai = append(userIdBocBai, userId)
			dem++
		}

	}
	return userIdBocBai
}

// list các user muốn rút gồm cả dealer
// kiểm tra sự tồn tại của userHand && là userHand || dealerHand
// exist => chia 1 lá

func (s *MatchState) DevideMoreCardForPlayer(userId []string, numberCard int) {
	if len(userId) > 0 {
		for _, userIdAddCard := range userId {
			listCard, err := s.deck.Deal(numberCard)
			if err != nil {
				log.Fatal("Xảy ra lỗi khi chia bài tại func devideOneCardForPlayer()!")
			}

			if userIdAddCard == s.playerIsDealer {
				if len(s.dealerHand.first) == 2 {
					s.dealerHand.first = append(s.dealerHand.first, listCard.Cards...)
				}
			} else {
				// khi nào thì check type shan ?
				// check cho player
				if s.IsPlayerHave_TypeShan(userIdAddCard) {
					continue
				}

				existUserHand := s.CheckExistUserHand(userIdAddCard)
				if existUserHand && len(s.userHands[userIdAddCard].first) == 2 {
					s.userHands[userIdAddCard].first = append(s.userHands[userIdAddCard].first, listCard.Cards...)
				}
			}
		}
	}

}

func (s *MatchState) CheckExistUserHand(userId string) bool {
	if _, exist := s.userHands[userId]; exist {
		return true
	}
	return false
}

func (s *MatchState) GetListUserId_playerNotFitBet() []string {
	userDontBetFit := []string{}
	for userId, player := range s.userBets {
		if player.First < MinBetAllowed {
			userDontBetFit = append(userDontBetFit, userId)
			delete(s.userBets, userId)
		}
	}
	return userDontBetFit
}

func (s *MatchState) DeletePlayerNotFitBet(userDontBetFit []string) {
	if len(s.userBets) > 0 && len(userDontBetFit) > 0 {
		// => xóa user khỏi userBet
		for _, userId_delete := range userDontBetFit {
			// xóa quyền dealer và xóa mức cược của playerIsDealer
			if s.playerIsDealer != "" && userId_delete == s.playerIsDealer {
				delete(s.userBets, userId_delete)
				s.playerIsDealer = ""
				s.SetInfoForDealer("")
			}

			delete(s.userBets, userId_delete)
			s.PlayingPresences.Remove(userId_delete) // => xóa user khỏi playingPrecence: có vì nó đại diện cho các player đang chơi game
		}
	} else {
		log.Println("Không có user nào đặt cược trong ván chơi này và không có player nào cần phải xóa!")
	}
}

func (s *MatchState) DeletedPlayerNotFitBet() {
	userDontBetFit := s.GetListUserId_playerNotFitBet()
	if len(userDontBetFit) > 0 {
		s.DeletePlayerNotFitBet(userDontBetFit)
	}
}
func (s *MatchState) PrintInfoOfBetInMatch() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets {
			fmt.Println("Userid = ", userBet, ", value = ", userBet.First)
		}
	} else {
		log.Fatal("Chưa có player nào đặt cược!")
	}
}

// xóa khi nào? khi ko còn đủ điều kiện minBet => xóa

// xử lý trường hợp bet của playerIsdealer không còn đủ điều kiện để chơi game
// xóa bet của player khỏi userBet
// xóa quyền làm dealer => set lại quyền làm dealer cho server
func (s *MatchState) DealerIsPlayer_reduceBet() {
	// mức cược còn phù hợp với trận đấu ko?
}

func (s *MatchState) RandDomPercentBet() int {
	arr_Bet := []int{1, 10, 15, 20, 50, 70, 90}
	rand.Seed(time.Now().Unix())
	randIndex := rand.Intn(len(arr_Bet))
	return arr_Bet[randIndex]
}

var presenceTest_aPerSonPlayMatch = []struct {
	userId     string
	myPresence MyPrecense
}{
	{"userTest1", MyPrecense{Chips: 10000, VipLevel: 2}},
}

func (s *MatchState) SetPresenceInMatch(players []pb.Player) {
	for _, player := range players {
		chipPlayer := ConvertWalletFromStrToInteger(player)
		myPrecence := MyPrecense{Chips: int64(chipPlayer), VipLevel: player.VipLevel}
		s.Presences.Put(player.Id, myPrecence)
	}
}

func (s *MatchState) RegisterDealer(lst_userID_player []string) {
	if s.playerIsDealer == "" && len(lst_userID_player) > 0 { // current server is  dealer
		for _, userId := range lst_userID_player {
			s.Player_RegisterDealer(userId)
		}
	} else { // player is dealer - player khác xin làm dealer thay thế
		for _, userId := range lst_userID_player {
			s.Set_PlayerCanBeDealer(userId)
		}
	}
}

// add bet for user
// func (s *MatchState) AddBetOfUserBet(v *pb.ShanGameBet) {
// 	// xem userId ko tồn tại trong userBet thì khởi tạo giá trị mới cho nó
// 	s.playerNotExits_inUserBets(v.UserId)

// 	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

// 	s.AddBet_inUserBets(v.UserId, v.Chips)

// 	s.SubstractMoney_inWallet_playingPresence(v.UserId, v.Chips)

// 	fmt.Println("UserID current sau khi tăng mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)
// }

func (s *MatchState) IsExitsPresence(userId string) bool {
	if _, exist := s.Presences.Get(userId); exist {
		return true
	}
	return false
}
