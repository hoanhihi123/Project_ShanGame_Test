package entity

import (
	"fmt"
	"log"
	"testing"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

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

/*
cac truong hop co the check

	server = dealer
	user = dealer
	user = join when user other join game

	deal card and manage card
	cal point and return player win

	end game
*/
func TestMatchState_hoan(t *testing.T) {

	fmt.Println("Preparing to play game...")
	s := NewMatchState(&MatchLabel{
		Open:     2,
		Bet:      5000,
		Code:     "test",
		Name:     "test_table",
		Password: "",
		MaxSize:  MaxPresences,
	})
	fmt.Printf("Thiết lập thông số cho 1 trận đấu...")

	// user request vào game
	fmt.Println("khởi tạo trận đấu .... ")

	fmt.Println("Các user join vào game vào precense of match ....")
	for _, precense := range default_presence {
		s.Presences.Put(precense.UserId, precense.MyPrecense)
	}

	// kiểm tra xem user có đủ điều kiện làm Dealer hay không ?
	// giả sử player muốn làm dealer
	// set trường hợp, server đang là dealer , player user5 xin làm dealer
	s.playerIsDealer = ""
	// 	trước khi vào trận đấu  - player xin làm dealer
	if s.playerIsDealer == "" { // server is  dealer
		// fmt.Println("chạy vào đăng ký làm dealer")
		// lấy chips của user muốn xin làm dealer
		s.Player_RegisterDealer("User5_dealer")
	} else { // player is dealer - player khác xin làm dealer thay thế
		// giả sử "User5_dealer" là player xin làm dealer
		s.set_PlayerCanBeDealer("User5_dealer")
	}

	fmt.Println("Server or Player is Dealer ? ", s.playerIsDealer, ", \t Pot = ", s.POT)
	fmt.Println("DealerHand current: ", s.dealerHand.userId)
	fmt.Println("UserHand current have size: ", len(s.userHands))

	s.Init()
	// set info of player with fake precencese
	fmt.Println("\nkhởi tạo các user join vào trận đấu để chơi .... ")

	// khi nào thì check maxPresence ?
	fmt.Println("Thêm các presence vào playingPrecense")
	s.addPresence_ToPlayingPrecense_InMatch()
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

		s.setAddBet_forPlayerAndDealer()
		fmt.Println("Xem thông tin các mức cược của các player - khi add mức cược ")
		if len(s.userBets) > 0 {
			for userId, player := range s.userBets {

				fmt.Println("Bet of ", userId, ", chip = ", player.First) // mong doi = 500
			}
		}

		s.setSubstractBet_forPlayerAndDealer()

		// lấy ra mức cược player đã đặt theo id tương ứng
		fmt.Println("\nXem thông tin các mức cược của các player - sau khi trừ đi mức cược ")
		if len(s.userBets) > 0 {
			for userId, player := range s.userBets {

				fmt.Println("Bet of ", userId, ", chip = ", player.First) // mong doi = 500
			}
		}
	} else {
		log.Fatal("Allow Bet chưa được cho phép cược!")
	}

	fmt.Println("\nuser request as player .... click deal ")

	s.deck.Shuffle()

	fmt.Println("kiểm tra các user nào không đưa ra mức cược => xóa khỏi userBet và playingPrecence...")
	s.DeletePlayerAtUserBetIfBalance_equalZero()
	fmt.Println("Các user còn lại sau khi kiểm tra mức cược có > 0 hay ko ? = ", len(s.userBets))

	// duyệt userBet
	for _, userBet := range s.userBets {
		fmt.Println("Userid = ", userBet, ", value = ", userBet.First)
	}

	fmt.Println("Chia bài .... cho các user đã đặt cược trong ván game .....")
	s.chiaBaiChoPlayerTuongUng()

	fmt.Println("\nSố lượng userHand : ", len(s.userHands))
	fmt.Println("\nLần lượt user : đưa ra lựa chọn bốc bài tiếp hay không ?")

	// kiểm tra bài của dealer = shan ko ?
	if len(s.userHands) > 0 {
		deckDealerHand_typeShan := s.checkDealerHand_haveTypeShan()
		fmt.Println("Dealer được bài shan không ? = ", deckDealerHand_typeShan)

		fmt.Println("Dealer ko được pok => user,dealer được bốc bài tiếp")
		if !deckDealerHand_typeShan {
			fmt.Println("Player bốc thêm bài...")
			s.devideCardForPlayer("User1", 1)

			fmt.Println("Dealer bốc thêm bài...")
			s.devideCardForPlayer(s.dealerHand.userId, 1)

			s.CalPointFor_Player_Dealer() // // kiểm tra user còn đủ điều kiện chơi tiếp ko ? - đã kiểm tra trong hàm ( cả user và player )

		} else {
			// dealer được bài pok => tính điểm và so sánh bài với các user luôn
			fmt.Println("Tiếp theo các player sẽ thực hiện so bài với dealer ...")
			s.CalPointFor_Player_Dealer() // // kiểm tra user còn đủ điều kiện chơi tiếp ko ? - đã kiểm tra trong hàm ( cả user và player )
		}
	} else {
		log.Fatal("Không có bộ bài nào để xét lose or win!")
	}

	// check finish game return cái gì mà khi user win trả về
	// số tiền mà user khi win thôi (dựa vào presence / playingprecense)
	// và khi thay đổi thì phần tiền (wallet of player đã được cập nhật hay chưa?)

	// fmt.Printf("====END GAME====\n%v", s.CalcGameFinish())
	s.getResultEndGame()

}
