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
	MinBetAllowed = 1
	MaxBetAllowed = 200
)

type MatchState struct {
	baseMatchState

	allowBet       bool
	allowInsurance bool
	allowAction    bool
	visited        map[string]bool // trường này dùng để làm gì
	userBets       map[string]*pb.ShanGamePlayerBet
	userLastBets   map[string]int64
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
		userLastBets:   make(map[string]int64, 0),
		userHands:      make(map[string]*Hand, 0), // những lá bài được chia cho player
		dealerHand:     &Hand{},                   // những lá bài được chia cho dealer
		currentTurn:    "",
		currentHand:    pb.ShanGameHandN0_SHANGAME_HAND_1ST, // bộ bài số mấy của player
		gameState:      pb.GameState_GameStateIdle,          // trạng thái trò chơi (start or end)
		updateFinish:   nil,                                 // update trạng thái trận đấu
		playerIsDealer: "",
		deck:           NewDeck(),
	}
}

// init info of user bets, user hands
func (s *MatchState) Init() {
	for k := range s.userBets {
		delete(s.userBets, k)
	}
	for k := range s.userHands {
		delete(s.userHands, k)
	}
	s.balanceResult = nil
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

// divide more card
func (s *MatchState) AddCards(cards []*pb.Card, userId string, handN0 pb.ShanGameHandN0) {
	if userId == "" {
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

func (s *MatchState) SetAllowInsurance(v bool) { s.allowInsurance = v }
func (s *MatchState) IsAllowInsurance() bool   { return s.allowInsurance }

func (s *MatchState) SetAllowAction(v bool) { s.allowAction = v }
func (s *MatchState) IsAllowAction() bool   { return s.allowAction }

func (s *MatchState) SetUpdateFinish(v *pb.ShanGameUpdateFinish) { s.updateFinish = v }
func (s *MatchState) GetUpdateFinish() *pb.ShanGameUpdateFinish  { return s.updateFinish }

func (s *MatchState) GetUserBetById(userId string) *pb.ShanGamePlayerBet { return s.userBets[userId] }

// check can bet ?
func (s *MatchState) IsCanBet(userId string, balance int64, bet *pb.ShanGameBet) bool {
	if bet.Chips+s.userBets[userId].First+s.userBets[userId].Insurance+s.userBets[userId].Second > int64(MaxBetAllowed*s.Label.Bet) {
		return false
	}
	if balance < bet.Chips {
		return false
	}
	return true
}

// check lại điều kiện, so sánh mức cược với tiền mà player có
// mức cược của user < min bet => true => user ko được tham gia match
func (m *MatchState) IsBetFitMatch(userId string) bool {
	return m.userBets[userId].First < int64(m.Label.Bet)
}

// set bet for user by id
func (s *MatchState) playerNotExits_inUserBets(userId string) {
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
	s.userBets[userId].First += chip
	s.userLastBets[userId] = s.userBets[userId].First
}

func (s *MatchState) SubstractBet_inUserBets(userId string, chip int64) {
	fmt.Println("Số chip hiện tại :", s.userBets[userId].First)
	fmt.Println("Số chip muốn giảm : ", chip)
	// add bet for user by id
	s.userBets[userId].First -= chip
	fmt.Println("Số chip trong bet sau khi giảm: ", s.userBets[userId].First)
	s.userLastBets[userId] = s.userBets[userId].First

}

func (s *MatchState) SubstractMoney_inWallet_playingPresence(userId string, chip int64) {
	if value, ok := s.PlayingPresences.Get(userId); ok {
		playingPresence := value.(MyPrecense)
		playingPresence.Chips -= chip // trừ đi tiền trong ví tạm của player tương ứng
		if !ok {
			// Xử lý trường hợp `User1` không tồn tại (nếu cần)
			log.Println(userId, " không tồn tại trong danh sách playingPresence")
		}
	}
}

func (s *MatchState) AddMoney_inWallet_playingPresence(userId string, chip int64) {
	if value, ok := s.PlayingPresences.Get(userId); ok {
		playingPresence := value.(MyPrecense)
		playingPresence.Chips += chip // trừ đi tiền trong ví tạm của player tương ứng
		if !ok {
			// Xử lý trường hợp `User1` không tồn tại (nếu cần)
			log.Println(userId, " không tồn tại trong danh sách Presences")
		}
	}
}

// add bet for user
func (s *MatchState) AddBetOfUserBet(v *pb.ShanGameBet) {
	// xem userId ko tồn tại trong userBet thì khởi tạo giá trị mới cho nó
	s.playerNotExits_inUserBets(v.UserId)

	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

	s.AddBet_inUserBets(v.UserId, v.Chips)

	s.SubstractMoney_inWallet_playingPresence(v.UserId, v.Chips)

	fmt.Println("UserID current sau khi tăng mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)
}

// substract bet for user
func (s *MatchState) SubstractBetOfUserBet(v *pb.ShanGameBet) {
	// xem userId ko tồn tại trong userBet thì khởi tạo giá trị mới cho nó
	s.playerNotExits_inUserBets(v.UserId)

	// check money substract > userBet truoc do ko ?
	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

	s.SubstractBet_inUserBets(v.UserId, v.Chips)
	s.AddMoney_inWallet_playingPresence(v.UserId, v.Chips)

	// check trường hợp sau khi trừ tiền
	// nếu các user sau khi trừ tiền và mức cược quay về = 0
	// chuyển user hiện tại vào user out game trong matchState
	// xóa userId của user đó trong userBet
	// xóa userId của user đó trong playingPrecence

	fmt.Println("UserID current sau khi giảm mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)
}

func (s *MatchState) GetBet(v *pb.ShanGameBet) int64 {
	return s.userBets[v.UserId].First
}

func (s *MatchState) IsCanInsuranceBet(userId string, balance int64) bool {
	return balance*2 >= s.userBets[userId].First
}

func (s *MatchState) InsuranceBet(userId string) int64 {
	s.userBets[userId].Insurance = s.userBets[userId].First / 2
	return s.userBets[userId].Insurance
}

func (s *MatchState) IsCanDoubleDownBet(userId string, balance int64, pos pb.ShanGameHandN0) bool {
	return balance >= s.userBets[userId].First
}

func (s *MatchState) DoubleDownBet(userId string, pos pb.ShanGameHandN0) int64 {
	r := int64(0)
	if pos == pb.ShanGameHandN0_SHANGAME_HAND_1ST {
		r = s.userBets[userId].First
		s.userBets[userId].First *= 2
	}
	return r
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

func (s *MatchState) Rebet(userId string) int64 {
	s.userBets[userId].First = s.userLastBets[userId]
	return s.userLastBets[userId]
}

func (s *MatchState) DoubleBet(userId string) int64 {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First >= MinBetAllowed*int64(s.Label.Bet) {
		r := s.userBets[userId].First
		s.userBets[userId].First *= 2
		s.userLastBets[userId] = s.userBets[userId].First
		return r
	} else if _, found := s.userLastBets[userId]; found {
		if _, found := s.userBets[userId]; !found {
			s.userBets[userId] = &pb.ShanGamePlayerBet{
				UserId:    userId,
				Insurance: 0,
				First:     0,
				Second:    0,
			}
		}
		s.userLastBets[userId] *= 2
		s.userBets[userId].First = s.userLastBets[userId]
		return s.userLastBets[userId]
	}
	return 0
}

func (s *MatchState) IsCanRebet(userId string, balance int64) bool {
	if _, found := s.userBets[userId]; found {
		return false
	}
	if _, found := s.userLastBets[userId]; !found || s.userLastBets[userId] > balance {
		return false
	}
	return true
}

func (s *MatchState) IsCanDoubleBet(userId string, balance int64) bool {
	if _, found := s.userBets[userId]; found {
		if s.userBets[userId].First > balance {
			return false
		} else {
			return true
		}
	} else if _, found := s.userLastBets[userId]; found && s.userLastBets[userId]*2 <= balance {
		return true
	}
	return false
}

func (s *MatchState) IsCanHit(userId string, pos pb.ShanGameHandN0) bool {
	return s.userHands[userId].PlayerCanDraw(pos)
}

// userBets : danh sách những người dùng đã đặt cược
// s.userBets[userId].First: tại thông số first
// nếu giá trị first > 0 , tức là số lần đặt cược của user đó
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
	for _, userBet := range s.userBets {
		if userBet.UserId != s.playerIsDealer {
		}
	}

	for _, userHand := range s.userHands {
		result := int(0)
		if s.dealerHand != nil {
			result = userHand.Compare(s.dealerHand) // result after compare each player - dealer

			moneyWallet := s.getMoneyIn_PresencesOfPlayer(userHand.userId)
			moneyPlay := s.getMoneyIn_PlayingPresencesOfPlayer(userHand.userId)
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

}

func (s *MatchState) getMoneyIn_PresencesOfPlayer(userId string) int {
	if value, exists := s.Presences.Get(userId); exists {
		if exists {
			presence := value.(MyPrecense)
			return int(presence.Chips)
		}
	}
	return 0
}

func (s *MatchState) getMoneyIn_PlayingPresencesOfPlayer(userId string) int {
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
	defer func() { s.userBets[userId].Insurance = 0 }()
	userBet := s.userBets[userId]
	r1 := s.userHands[userId].Compare(s.dealerHand)
	first := &pb.ShanGameBetResult{
		BetAmount: userBet.First,
		WinAmount: 0,
		Total:     userBet.First,
	}

	if first.BetAmount > 0 {
		first.IsWin = int32(r1)
		if r1 > 0 {
			first.WinAmount = first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
		} else if r1 < 0 {
			first.WinAmount = -first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
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

// xác định ai thắng thua
func (s *MatchState) CalPointFor_Player_Dealer() {
	fmt.Println("================================ Tính tiền ==================")
	fmt.Println("Số lượng người dùng: ", len(s.userHands))
	fmt.Printf("Thông tin của dealerHand current : %+v", s.dealerHand)

	// list userId is win
	userId_result_Win := map[string]int64{}

	if len(s.userHands) > 0 {
		fmt.Println("\n\nSo sánh bài xem player or dealer win / lose ....")
		for _, userHand := range s.userHands {
			result := int(0)
			if s.dealerHand != nil {
				result = userHand.Compare(s.dealerHand) // result after compare each player - dealer
			}
			tiLeAn_dealer := s.dealerHand.GetTiLeThangThuaPlayer() // get ti le thang, thua
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			fmt.Println("\nTỉ lệ ăn dealer : ", tiLeAn_dealer)
			fmt.Println("Tỉ lệ ăn player : ", tiLeAn_player)

			if result == 1 { // player win, dealer lose
				fmt.Println("Player: " + userHand.userId + " - WIN ^_^")

				if s.playerIsDealer != "" { // player is dealer => cần tính lại, sau khi lấy được danh sách các player_win
					fmt.Println("Trường hợp playerIsDealer, nên tính tiền sẽ được tính sau....")
					userId_result_Win[userHand.userId] = int64(tiLeAn_player)
				} else {
					// option: dealer is server => chỉ cộng tiền cho user luôn
					fmt.Println("Trường hợp dealer là server, cộng tiền cho user luôn")
					chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)
					fmt.Println("Tiền hiện tại trong ví của player ", userHand.userId, ", = ", chips_inWalletOfPlayer)
					fmt.Println("Mức cược của user = ", s.GetBetOfUser_byID(userHand.userId))
					fmt.Println("Tỉ lệ ăn player = ", int64(tiLeAn_player))
					moneyPlayerWin := s.GetBetOfUser_byID(userHand.userId) * int64(tiLeAn_player)
					fmt.Println("Tiền thắng của player = ", moneyPlayerWin)
					tienHoPlayer := int64((float64(s.GetTiLeTienHo_User(userHand.userId)) / 100) * float64(moneyPlayerWin))

					updateChipPlayerWin := (chips_inWalletOfPlayer + (moneyPlayerWin - tienHoPlayer))
					fmt.Println("Số tiền thắng cần cập nhật trong ví player = ", updateChipPlayerWin)
					s.CalculatorMoneyForUserWin(1, userHand.userId, int64(tiLeAn_player), updateChipPlayerWin, s.GetBetOfUser_byID(userHand.userId))
				}

			} else if result == -1 { // dealer win , player lose
				fmt.Println("Dealer : " + s.dealerHand.userId + " - WIN ^_^")

				fmt.Println("Mức tiền cược của dealer = ", s.userBets[s.playerIsDealer].First)
				moneyDealerWin := s.GetBetOfUser_byID(s.dealerHand.userId) * int64(tiLeAn_dealer)
				fmt.Println("Số tiền mà Dealer win = ", moneyDealerWin)

				// lấy tiền trong wallet of player
				chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)
				fmt.Println("Số tiền trong ví hiện tại của dealer = ", s.POT)

				// check player có đủ tiền trả cho dealer không ?
				// check (tổng tiền trong wallet of user + tiền user đã cược)
				if (chips_inWalletOfPlayer + s.GetBetOfUser_byID(userHand.userId)) < moneyDealerWin { // dealer win //  tổng tiền player không đủ trả cho dealer

					s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), 0, 0)

				} else { // dealer win (tổng tiền của player đủ trả cho dealer)
					// sẽ có trường hợp  trong ví đủ tiền
					// ngược lại thì cần lấy tiền từ ván cược => thanh toán
					if chips_inWalletOfPlayer >= moneyDealerWin { // dealer win
						s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), (chips_inWalletOfPlayer - moneyDealerWin), s.GetBetOfUser_byID(userHand.userId))
					} else {
						// trường hợp còn lại
						// dealer win, player không đủ tiền trong ví để trả server, cần lấy tiền cược để trả server
						s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), ((chips_inWalletOfPlayer + s.GetBetOfUser_byID(userHand.userId)) - moneyDealerWin), 0)

					}

					// xóa player khỏi match
					s.PlayingPresences.Remove(userHand.userId)
				}
			} else {
				fmt.Println("Player:", userHand.userId, " and Dealer:", s.dealerHand.userId, " hòa nhau")
			}
		}
	} else {
		log.Fatal("UserHands không có giá trị nào để thực hiện so sánh bài!")
	}

	if len(userId_result_Win) > 0 { // player win && dealer is player
		isResultCheckSumMoney := s.Check_dealerIsPlayer_enoughMoney(userId_result_Win)
		fmt.Println("Dealer có đủ tiền trả các player win không ? = ", isResultCheckSumMoney)
		// player is dealer
		if isResultCheckSumMoney { // dealer_is_player have enough money for pay player_win
			for _, userHand := range s.userHands {
				betOfPlayerInMatch := s.GetBetOfUser_byID(userHand.userId)
				tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
				chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)

				fmt.Println("\nSố tiền player_", userHand.userId, " đặt cược = ", betOfPlayerInMatch)
				fmt.Println("Tỉ lệ ăn của player = ", tiLeAn_player)
				fmt.Println("Số tiền hiện có trong ví của player_ ", userHand.userId, " = ", chips_inWalletOfPlayer)

				// xem danh sach user = userId_result ?
				for key_userResult := range userId_result_Win {
					if key_userResult == userHand.userId {
						// tiền ăn của player
						moneyWinPlayer := betOfPlayerInMatch * int64(tiLeAn_player)
						fmt.Println("Tiền thắng thực tế của player = ", moneyWinPlayer)
						fmt.Println("Tỉ lệ tiền hồ của user = ", s.GetTiLeTienHo_User(userHand.userId))
						// tính tiền hồ mà server nhận
						tienHoPlayer := int64((float64(s.GetTiLeTienHo_User(userHand.userId)) / 100) * float64(moneyWinPlayer))
						fmt.Println("Tiền hồ player phải trả cho sys = ", tienHoPlayer)

						// cập nhật tiền trong ví của player win
						updateMoneyPlayerWin := (moneyWinPlayer + chips_inWalletOfPlayer - tienHoPlayer)
						fmt.Println("Số tiền trong ví sau khi player win = ", updateMoneyPlayerWin)

						s.CalculatorMoneyForUserWin(1, userHand.userId, int64(tiLeAn_player),
							updateMoneyPlayerWin, betOfPlayerInMatch)
					}
				}
			}
		} else { // dealer không đủ tiền trả cho các player
			fmt.Println("\n\nDealer không đủ tiền trả cho các player ....")
			// tính tỉ lệ thắng của từng user và lấy tiền từ wallet of dealer trả lần lượt cho các player
			sumMoney_DealerPaid_Player := s.sumMoneyDealer_needToPayPlayer(userId_result_Win)
			sumMoney_DealerCurrent := s.POT
			moneyPlayerWin_fact := int64(0)
			moneyPlayerWin_tiLe := int64(0)
			tienHoPlayer := int64(0)

			for _, userHand := range s.userHands {
				// lấy tỉ lệ ăn player
				betOfPlayerInMatch := s.GetBetOfUser_byID(userHand.userId)
				tiLeAn_player := userHand.GetTiLeThangThuaPlayer()

				// xem danh sach user = userId_result ?
				for key_userResult := range userId_result_Win {
					// lấy ra các user thắng dealer_isPlayer
					if key_userResult == userHand.userId {
						// tính tiền thắng cho từng player, thiết lập lại tiền cho player
						moneyPlayerWin_fact = int64(tiLeAn_player) * betOfPlayerInMatch
						fmt.Println("Tiền thắng thực tế của player = ", moneyPlayerWin_fact)

						// tiền userWin = (userWinFact / tổng tiền phải được nhận) * tổng tiền hiện còn của dealer
						moneyPlayerWin_tiLe = (moneyPlayerWin_fact / sumMoney_DealerPaid_Player) * sumMoney_DealerCurrent
						fmt.Println("Tiền thắng chia theo tỉ lệ của player = ", moneyPlayerWin_tiLe)

						// tính tiền hồ mà server nhận
						tienHoPlayer = int64((float64(s.GetTiLeTienHo_User(userHand.userId)) / 100) * float64(moneyPlayerWin_tiLe))
						fmt.Println("Tiền hồ player phải trả cho server = ", tienHoPlayer)
						s.moneyOfServer += tienHoPlayer

						fmt.Println("Tiền mà player thay đổi = ", (moneyPlayerWin_tiLe - tienHoPlayer))
						// update wallet of Player
						// set wallet of player = 0
						s.SetUserChipsInWallet(userHand.userId, moneyPlayerWin_tiLe-tienHoPlayer)

						// player is dealer
						s.POT -= moneyPlayerWin_tiLe
						fmt.Println("Số tiền trong POT còn lại hiện tại = ", s.POT)

						// kiểm tra xem dealer còn đủ điều kiện làm dealer hay không ?
						if s.POT < int64(s.Label.Bet) {
							s.SetUserChipsInWallet(s.dealerHand.userId, 0)
							s.playerIsDealer = ""
							s.dealerHand = &Hand{} // nơi chứa bộ bài của dealer (là player/server đều sẽ set value vào đây)
						}
					}
				}
			}
		}
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

// check sum money dealer_is_player need to pay for player
func (s *MatchState) Check_dealerIsPlayer_enoughMoney(userId_result_Win map[string]int64) bool {
	sum_moneyDealerNeed_paid := int64(0)
	if len(userId_result_Win) > 0 {
		for _, userHand := range s.userHands {
			s.gameState = pb.GameState_GameStateReward // trận đấu status: được tính điểm
			result := userHand.Compare(s.dealerHand)
			// lấy tỉ lệ ăn player
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			// xem danh sach user = userId_result ?
			for key_userResult := range userId_result_Win {
				if result == -1 && key_userResult == userHand.userId {
					// tinh tong tien cua dealer
					sum_moneyDealerNeed_paid += int64(tiLeAn_player) * s.userBets[key_userResult].First
				}
			}
		}
	}

	if sum_moneyDealerNeed_paid > s.POT {
		return false
	}
	return true
}

// player win dealer, but dealer not enough money to pay for each player win
// sum money dealer_is_player need to pay for player
func (s *MatchState) sumMoneyDealer_needToPayPlayer(userId_result_Win map[string]int64) int64 {
	sum_moneyDealerNeed_paid := int64(0)
	if len(userId_result_Win) > 0 {
		for _, userHand := range s.userHands {
			// lấy tỉ lệ ăn player
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			// xem danh sach user = userId_result ?
			for key_userResult := range userId_result_Win {
				if key_userResult == userHand.userId {
					// tinh tong tien cua dealer
					sum_moneyDealerNeed_paid += int64(tiLeAn_player) * s.userBets[userHand.userId].First
				}
			}
		}
	}

	return sum_moneyDealerNeed_paid
}

func (s *MatchState) GetBetOfUser_byID(userId string) int64 { return s.userBets[userId].First }

// -1 player lose, 1 player win, 0 hoa nhau
func (s *MatchState) CalculatorMoneyForUserWin(
	statusMatch int, userID_player string, tiLeAn_player int64,
	setChipPlayer int64, setBetOfPlayer int64) {

	moneyBet_dealer := s.GetBetOfUser_byID(s.dealerHand.userId) // mức tiền cược dealer
	moneyBet_player := s.GetBetOfUser_byID(userID_player)       // mức tiền cược player

	tiLeAn_dealer := s.dealerHand.GetTiLeThangThuaPlayer() // get ti le thang, thua
	// tiLeAn_player := s.userHands.GetTiLeThangThuaPlayer()

	moneyDealerWin := moneyBet_dealer * int64(tiLeAn_dealer)
	moneyPlayerWin := moneyBet_player * int64(tiLeAn_player)

	tienHoPlayer := int64((float64(s.GetTiLeTienHo_User(userID_player)) / 100) * float64(moneyPlayerWin))

	// lấy tiền trong wallet of player
	chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userID_player) // get chip from playingPresence

	if statusMatch == -1 { // dealer win
		// tiền thắng của dealer chưa chắc = mức cược dealer * tỉ lệ thắng dealer , vì player chưa chắc đủ tiền trả
		if moneyBet_player+chips_inWalletOfPlayer < moneyDealerWin {
			moneyDealerWin = moneyBet_player + chips_inWalletOfPlayer
		}

		// moneyDealerWin = (chips_inWalletOfPlayer + moneyBet_player) * tiLeAn_player // all money player lose have
		// set chip in wallet of player = 0
		s.SetUserChipsInWallet(userID_player, setChipPlayer) // set chip at playingPresence

		// set bet of player in match = 0
		s.userBets[userID_player].First = setBetOfPlayer

		// xóa player khỏi match
		if moneyBet_player < int64(s.Label.Bet) {
			s.PlayingPresences.Remove(userID_player)
		}
		// chưa set trường hợp nếu dealer lose => bị trừ tiền, có thể ko đủ điều kiện làm dealer nữa

		// moneyDealerWin := moneyBet_dealer * int64(tiLeAn_dealer)
		// moneyPlayerWin := moneyBet_player * int64(tiLeAn_player)
		fmt.Println("Money Dealer Win = ", moneyDealerWin)

		// tính tiền hồ mà server nhận
		tienHoDealer := int64((float64(s.GetTiLeTienHo_User(s.dealerHand.userId)) / 100) * float64(moneyDealerWin))
		fmt.Println("\nDealer win, tiền hồ mà dealer phải trả = ", tienHoDealer)
		s.moneyOfServer += int64(tienHoDealer)

		// set money condition Dealer is player
		if s.playerIsDealer != "" {
			s.POT += (moneyDealerWin - int64(tienHoDealer))
			fmt.Println("\n Tiền của dealer sau khi thắng = ", s.POT, "và dealer có id = ", s.playerIsDealer)
		} else { // dealer is server
			s.moneyOfServer += (moneyDealerWin - int64(tienHoDealer))
			fmt.Println("\n Tiền của dealer là server sau khi thắng = ", s.moneyOfServer)
		}
	} else if statusMatch == 1 { // player win

		// fmt.Println("Money player win = ", moneyPlayerWin)
		// - server ( chỉ cộng tiền cho player thông thường + tính tiền hồ cho server )
		// - player is dealer ( cộng tiền cho player + trừ tiền của player từ POT + tính tiền hồ cho server )

		// cộng tiền hồ cho server
		s.moneyOfServer += int64(tienHoPlayer)
		fmt.Println("Tiền hồ mà player phải trả server = ", tienHoPlayer)

		// set mức cược của player trong mức cược
		s.userBets[userID_player].First = setBetOfPlayer

		// set mức chip mà player thắng + tiền trong ví vào ví của player
		s.SetUserChipsInWallet(userID_player, setChipPlayer)

		// trừ tiền của server nếu player is dealer
		if s.playerIsDealer != "" { // player is dealer , dealer is server thì ko bị trừ tiền
			s.POT -= moneyPlayerWin
			fmt.Println("Dealer là server, với id = ", s.playerIsDealer, " , sau khi trừ tiền còn = ", s.POT)
		}
	}
}

func (s *MatchState) chiaBaiChoPlayerTuongUng() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets { // lấy danh sách userBet - lấy ra userID tham gia
			fmt.Println("Chia bài cho user hiện tại có id: " + userBet.UserId)
			listCard_chia2La, err := s.deck.Deal(2) // mỗi userBet - lấy ra 2 lá bài trong bộ bài

			if err != nil {
				log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
			}

			fmt.Printf("Danh sách lá bài được rút: %+v", listCard_chia2La.Cards)
			// chưa check trường hợp userBet là userHand của PlayerIsDealer
			// User1 tại sao set value lại lỗi do chưa khởi tạo giá trị từ trước ?

			// chia bài cho dealer
			if userBet.UserId == s.playerIsDealer {
				// trường hợp chưa được chia lá bài nào
				fmt.Println("\nChia bài cho dealer, với id = " + userBet.UserId)
				if len(s.dealerHand.first) == 0 { // dealerhand chưa được khởi tạo và thêm giá trị nào
					s.dealerHand = &Hand{
						userId: userBet.UserId,
						first:  listCard_chia2La.Cards,
					}
				} else if len(s.dealerHand.first) == 2 {
					listCard_addOneCard, err := s.deck.Deal(1)
					if err != nil {
						log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
						continue
					} else {
						s.dealerHand.first = append(s.dealerHand.first, listCard_addOneCard.Cards...)
					}
				}
				fmt.Printf("Bộ bài của dealer sau khi chia bài: %+v\n", s.dealerHand.first)
			} else { // chia bài cho player
				fmt.Println("\nChia bài cho player, với id = " + userBet.UserId)
				if len(s.userHands) > 0 {
					isCheckUserId_existInUserHand := true  // check userId đã có trong userHand chưa
					for _, userHand := range s.userHands { // tại sao nó chạy vào đây là lỗi ?
						if len(userHand.first) == 2 && userHand.userId == userBet.UserId {
							isCheckUserId_existInUserHand = false
							listCard_addOneCard, err := s.deck.Deal(1)
							if err != nil {
								log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
								continue
							} else {
								userHand.first = append(userHand.first, listCard_addOneCard.Cards...)
							}
							break
						}

					}

					if isCheckUserId_existInUserHand { // userId chưa tồn tại trong UserHand
						// set thông tin bài chia được vào userHands
						s.userHands[userBet.UserId] = &Hand{
							userId: userBet.UserId,
							first:  listCard_chia2La.Cards,
						}
					}
				} else { // trường hợp userHand chưa có chứa thông tin của bất kỳ ai => thêm mới 1 userHand tại userId hiện tại
					s.userHands[userBet.UserId] = &Hand{
						userId: userBet.UserId,
						first:  listCard_chia2La.Cards,
					}
				}
				fmt.Printf("Bộ bài của player sau khi chia bài: %+v\n", s.userHands[userBet.UserId].first)
			}
		}
	} else {
		fmt.Println("Message thông báo: Chưa có user nào đặt cược vào ván chơi!")
		return
	}
}

func (s *MatchState) Player_RegisterDealer(userId_param string) {
	if value, ok := s.Presences.Get(userId_param); ok {
		// fmt.Printf("Kiểu dữ liệu thực tế của value trả về : %T", value)

		presence, ok2 := value.(MyPrecense)
		if !ok2 {
			fmt.Println("Presence not have type MyPrecence...")
		} else {
			// player
			if presence.Chips > int64(s.Label.Bet*10) { // so sánh với mức cược tối thiểu của trận đấu
				s.playerIsDealer = userId_param
				s.POT = presence.Chips // set banker = tổng tiền trong ví của player đang có
				s.dealerHand = &Hand{
					userId: userId_param,
				}
			} else { // server is dealer
				if value, ok := s.Presences.Get(""); ok {
					server_presence := value.(MyPrecense)

					s.playerIsDealer = ""
					s.POT = server_presence.Chips // set chips của server đặt cược vào POT
					fmt.Println("Chip của server = ", server_presence.Chips)

					s.dealerHand = &Hand{
						userId: "",
					}
				}
			}
		}
	}

}

func (s *MatchState) getChip_PresenceById(userId string) int64 {
	chip := int64(0)
	if value, ok := s.Presences.Get(userId); ok {
		precence := value.(MyPrecense)
		chip = precence.Chips
	}
	return chip
}

func (s *MatchState) set_PlayerCanBeDealer(idPlayerWantBeDealer string) {
	// lấy mức chip của player đang là dealer
	chip_playerIsDealer := s.getChip_PresenceById(s.playerIsDealer)

	// lấy mức chip của player khác muốn xin làm dealer thay thế
	chip_player2 := s.getChip_PresenceById(idPlayerWantBeDealer)

	// check điều kiện đủ để player khác xin làm dealer - thay thế dealer current
	if chip_player2 > chip_playerIsDealer && chip_player2 > int64(s.Label.Bet) {
		s.playerIsDealer = idPlayerWantBeDealer
		s.POT = chip_player2 // set banker = tổng tiền trong ví mà dealer hiện có
		s.dealerHand = &Hand{
			userId: idPlayerWantBeDealer,
		}
	}
}

func (s *MatchState) addPresence_ToPlayingPrecense_InMatch() {
	// add dữ liệu từ precense vào playingprecense
	if s.Presences.Size() > 0 {
		for _, key := range s.Presences.Keys() {
			key, ok := key.(string)
			value, ok := s.Presences.Get(key)
			presence := value.(MyPrecense)
			if ok {
				chipsUser := presence.Chips
				if chipsUser >= int64(s.Label.Bet)*10 {
					// khi nào thì ko add server từ precen vào playing , tất cả trường hợp
					if key == "" {
						continue
					}
					if s.PlayingPresences.Size() <= 7 {
						s.PlayingPresences.Put(key, presence)
					}
				}
			}

		}
	}
}

func (s *MatchState) setAddBet_forPlayerAndDealer() {

	// duyệt danh sách playing precense => set mức đặt cược theo % đặt cược random
	fmt.Println("Set mức cược cho các player .... ")
	for _, key := range s.PlayingPresences.Keys() {
		userId, ok := key.(string)
		value, ok := s.PlayingPresences.Get(key)
		player := value.(MyPrecense)
		if ok {
			percentBet := int64(s.randDomPercentBet())
			fmt.Println("Percent Bet : ", percentBet)
			chipsUserBet := player.Chips * (100 - percentBet) / 100
			fmt.Println("UserId = ", userId, ", Đặt cược = ", chipsUserBet, " tiền trong ví trước đó = ", player.Chips)
			// if userId này chưa tồn tại => khởi tạo 1 userBet rỗng
			s.playerNotExits_inUserBets(userId)

			s.AddBet_inUserBets(userId, chipsUserBet)

			s.SubstractMoney_inWallet_playingPresence(userId, chipsUserBet)

		}

	}

	if s.playerIsDealer == "" {
		fmt.Println("Set mức cược cho dealer nếu dealer là server")
		s.playerNotExits_inUserBets("")
		s.AddBet_inUserBets("", 10000)
	}

}

func (s *MatchState) setSubstractBet_forPlayerAndDealer() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets {
			chipsUserBet := s.userBets[userBet.UserId].First * (100 - int64(s.randDomPercentBet())) / 100
			fmt.Println("UserId = ", userBet.UserId, ", Đặt cược = ", chipsUserBet, " tiền đặt cược trước đó = ", userBet.First)

			s.SubstractBet_inUserBets(userBet.UserId, chipsUserBet)
			s.AddMoney_inWallet_playingPresence(userBet.UserId, chipsUserBet)
		}
	}
}

func (s *MatchState) checkDealerHand_haveTypeShan() bool {
	dealerPoint, dealerHand := s.dealerHand.Eval() // tính điểm cho dealer và lấy ra type của nó
	return dealerPoint > 0 && dealerHand == pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN
}

func (s *MatchState) isPlayerHave_TypeShan(userIdPlayer string) bool {
	playerPoint, playerTypeHand := s.userHands[userIdPlayer].Eval()
	return playerPoint >= 0 && playerTypeHand != pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN && len(s.userHands[userIdPlayer].first) <= 3
}

// func chia thêm bài cho user nào , số lượng lá bài chia
func (s *MatchState) devideCardForPlayer(userId string, numberCard int) {
	// kiểm tra sự tồn tại của userId trong userHand
	if _, existsUserId := s.userHands[userId]; existsUserId {
		listCard, err := s.deck.Deal(numberCard)
		// check điều kiện khi nào thì player hợp lệ để bốc bài tiếp
		if err == nil && s.isPlayerHave_TypeShan(userId) {
			fmt.Println("Trước khi set giá trị cho user1:", s.userHands[userId].first)
			s.userHands[userId].first = append(s.userHands[userId].first, listCard.Cards...)
		}
		fmt.Printf("Bộ bài sau khi rút lần 2 của user 1: \n%+v", s.userHands[userId].first)
	} else {
		log.Println("Player : ", userId, " này không có trong_UserHand, các player tham gia trận đấu ")
	}

}

func (s *MatchState) DeletePlayerAtUserBetIfBalance_equalZero() {

	userIdHaveBalance_zero := []string{}
	for userId, player := range s.userBets {
		if player.First == 0 {
			userIdHaveBalance_zero = append(userIdHaveBalance_zero, userId)
			delete(s.userBets, userId)
		}
	}

	// => xóa user khỏi userBet
	for _, userId_delete := range userIdHaveBalance_zero {
		// xóa mức cược của server khỏi trận đấu này khi, player là server
		if s.playerIsDealer != "" {
			delete(s.userBets, "")
		}

		delete(s.userBets, userId_delete)
		s.PlayingPresences.Remove(userId_delete) // => xóa user khỏi playingPrecence: có vì nó đại diện cho các player đang chơi game
	}

}

func (s *MatchState) randDomPercentBet() int {
	arr_Bet := []int{1, 10, 15, 20, 50, 70, 90}
	rand.Seed(time.Now().Unix())
	randIndex := rand.Intn(len(arr_Bet))
	return arr_Bet[randIndex]
}

func (s *MatchState) setBetForServerIsDealer() {
	if s.playerIsDealer == "" {

	}
}
