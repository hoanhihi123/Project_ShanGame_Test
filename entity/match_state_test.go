package entity

import (
	"fmt"
	"log"
	"testing"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

/*
cac truong hop co the check

	server = dealer
	user = dealer
	user = join when user other join game

	deal card and manage card
	cal point and return player win

	end game
*/

var default_presence = []struct {
	UserId     string
	MyPrecense MyPrecense
}{
	{"", MyPrecense{Chips: 1000000, VipLevel: 3}},
	{"User1", MyPrecense{Chips: 100000, VipLevel: 1}},
	{"User2", MyPrecense{Chips: 2000000, VipLevel: 2}},
	{"User3", MyPrecense{Chips: 1500000, VipLevel: 5}},
	{"User4", MyPrecense{Chips: 150000, VipLevel: 6}},
	{"User5_dealer", MyPrecense{Chips: 45000, VipLevel: 7}},
	{"User6", MyPrecense{Chips: 900000, VipLevel: 3}},
	{"User7", MyPrecense{Chips: 260000, VipLevel: 4}},
	{"User8", MyPrecense{Chips: 150000, VipLevel: 9}},
}

var test_mul_Player_playWithServer = []pb.Player{
	{Id: "User1", VipLevel: 2, Wallet: "200000"},
	{Id: "User2", VipLevel: 4, Wallet: "150000"},
	{Id: "User3", VipLevel: 9, Wallet: "400000"},
	{Id: "User4", VipLevel: 6, Wallet: "220000"},
	{Id: "User5", VipLevel: 1, Wallet: "100000"},
	{Id: "User6", VipLevel: 3, Wallet: "100000"},
	{Id: "User7", VipLevel: 5, Wallet: "600000"},
	{Id: "User8", VipLevel: 7, Wallet: "800000"},
}

var test_1Player_playWithServer = []pb.Player{
	{Id: "User1", VipLevel: 2, Wallet: "90000"},
	{Id: "User2", VipLevel: 4, Wallet: "125000"},
	// {Id: "User3", VipLevel: 1, Wallet: "81000"},
	// {Id: "User4", VipLevel: 6, Wallet: "110000"},
	// {Id: "User5", VipLevel: 3, Wallet: "25000"},
	// {Id: "User6", VipLevel: 7, Wallet: "30000"},
	// {Id: "User7", VipLevel: 7, Wallet: "30000"},
	// {Id: "User8", VipLevel: 7, Wallet: "30000"},
}

func TestMatchState_MatchLoop(t *testing.T) {

	s := NewMatchState(&MatchLabel{
		Open: 2,
		// Bet:      5000, // mức cược của dealer is server
		Code:     "test",
		Name:     "test_table",
		Password: "",
		MaxSize:  MaxPresences,
	})
	// xóa thông tin của các user ko cần thiết tại đây: userHand, dealerHand, set lại mức cược

	s.Label.Bet = 5000
	// user request vào game
	fmt.Printf("Thiết lập thông số cho 1 trận đấu...")
	fmt.Println("khởi tạo trận đấu .... ")
	fmt.Println("Các user join vào game vào precense of match ....")
	if len(test_1Player_playWithServer) > 0 {
		fmt.Println("run. ..")
		s.SetPresenceInMatch(test_1Player_playWithServer)
	} else {
		log.Fatal("Không có player nào hiện tại!")
	}

	soLaLap := int(0)

	for len(s.Presences.Keys()) > 0 {
		soLaLap++
		// player chỉ muốn join để chơi với server
		// những thông số nào cần set cho dealer? userBet, dealerHand,
		lst_userID_registerDealer := []string{}
		s.playerIsDealer = ""
		// 	trước khi vào trận đấu  - player xin làm dealer
		fmt.Println("Sys xử lý nếu có player đăng ký làm dealer, ngược lại set các thông số cho server là dealer...")
		s.RegisterDealer(lst_userID_registerDealer)

		fmt.Println("DealerHand current: ", s.dealerHand.userId)
		fmt.Println("UserHand current have size: ", len(s.userHands))
		fmt.Println("Server or Player is Dealer ? ", s.playerIsDealer, ", \t Pot = ", s.POT)

		s.Init()
		// set info of player with fake precencese
		fmt.Println("\nkhởi tạo các user join vào trận đấu để chơi .... ")

		// khi nào thì check maxPresence ?
		fmt.Println("Thêm các presence vào playingPrecense")
		s.AddPresence_ToPlayingPrecense_InMatch()
		fmt.Println("Số lượng player được thêm vào _ playingPrecence: ", s.PlayingPresences.Size())

		// them muc cuoc cho user ( chon thong thuong, + - )
		s.gameState = pb.GameState_GameStatePreparing
		s.allowBet = true
		fmt.Println("Các user chọn mức tiền cược để tham gia trận đấu .... ")
		fmt.Println("Thiết lập trạng thái match = ", s.gameState)
		fmt.Println("Thiết lập trạng thái cho phép cược = ", s.allowBet)

		if s.allowBet {
			fmt.Println("Thiết lập mức cược của PlayerIsDealer ... ")
			fmt.Println("Thiết lập mức cược cho các player ... ")

			s.SetAddBet_forPlayerAndDealer()
			fmt.Println("Xem thông tin các mức cược của các player - khi add mức cược ")
			s.PrintInfoOfBetInMatch()

			s.SetSubstractBet_forPlayerAndDealer()
			// lấy ra mức cược player đã đặt theo id tương ứng
			fmt.Println("\nXem thông tin các mức cược của các player - sau khi trừ đi mức cược ")
			s.PrintInfoOfBetInMatch()
		} else {
			log.Fatal("Trạng thái allowBet trong trận đấu chưa được thiết lập!")
		}

		fmt.Println("\nuser request as player .... click deal ")
		// 		nếu s.Dealt < (1 dealer + n player còn lại )* 3 lá
		// => khởi tạo bộ bài mới => cho ng chơi chơi tiếp
		checkSLLaBai := (1 + len(s.PlayingPresences.Keys())*3)
		fmt.Println("checkSLLaBai = ", checkSLLaBai)
		fmt.Println("s.deck.Dealt = ", s.deck.Dealt)
		if checkSLLaBai < s.deck.Dealt {
			s.deck = NewDeck()
		}
		fmt.Println("s.deck.Dealt = ", s.deck.Dealt)
		s.deck.Shuffle()

		fmt.Println("kiểm tra các user nào không đưa ra mức cược => xóa khỏi userBet và playingPrecence...")
		s.DeletedPlayerNotFitBet()
		fmt.Println("Các user còn lại sau khi kiểm tra mức cược có > mức cược tối thiểu ? \nSố lượng userBet còn đặt cược = ", len(s.userBets))

		// duyệt userBet
		s.PrintInfoOfBetInMatch()

		fmt.Println("Chia bài .... cho các user đã đặt cược trong ván game .....")
		s.ChiaBaiChoPlayerTuongUng()

		fmt.Println("\nSố lượng userHand : ", len(s.userHands))
		fmt.Println("\nLần lượt user : đưa ra lựa chọn bốc bài tiếp hay không ?")

		// gia su các user outgame
		userIdOutGame := []string{"User1", "User2"}
		// khởi tạo và gán giá trị vào leavePresence
		for _, userIdOut := range userIdOutGame {
			s.LeavePresences.Put(userIdOut, userIdOut) // danh sách các player đang chơi thì out game
		}

		// kiểm tra bài của dealer = shan ko ?
		if len(s.userHands) > 0 {
			deckDealerHand_typeShan := s.CheckDealerHand_haveTypeShan()
			fmt.Println("Dealer được bài shan không ? = ", deckDealerHand_typeShan)

			fmt.Println("Dealer ko được pok => user,dealer được bốc bài tiếp")
			if !deckDealerHand_typeShan {
				fmt.Println("Player bốc thêm bài...")
				// userId_rutThemBai := []string{"User1", "User3", "User6"}
				userId_rutThemBai := s.GetRandomPlayerBocBai()

				if len(userId_rutThemBai) > 0 {
					s.DevideMoreCardForPlayer(userId_rutThemBai, 1)
				}

				s.CalMoney_whoWin() // kiểm tra user còn đủ điều kiện chơi tiếp ko ? - đã kiểm tra trong hàm ( cả user và player )

			} else {
				// dealer được bài pok => tính điểm và so sánh bài với các user luôn
				s.CalMoney_whoWin() // kiểm tra user còn đủ điều kiện chơi tiếp ko ? - đã kiểm tra trong hàm ( cả user và player )
			}
		} else {
			log.Fatal("Không có bộ bài nào để xét lose or win!")
		}

		// fmt.Printf("====END GAME====\n%v", s.CalcGameFinish())
		s.getResultEndGame()

		fmt.Println("Số lần lặp  = ", soLaLap)
		fmt.Println("Presence current = ", len(s.Presences.Keys()))
	}

}
