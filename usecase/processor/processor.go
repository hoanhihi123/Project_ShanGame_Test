package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	pb "github.com/nakamaFramework/cgp-common/proto"
	"google.golang.org/protobuf/encoding/protojson"

	"shangame-module/entity"
	"shangame-module/usecase/engine"

	"github.com/heroiclabs/nakama-common/runtime"
)

type Processor struct {
	*BaseProcessor
	turnBaseEngine *TurnBaseEngine
}

func NewMatchProcessor(
	marshaler *protojson.MarshalOptions,
	unmarshaler *protojson.UnmarshalOptions,
	engine engine.UseCase,
) IProcessor {
	return &Processor{
		NewBaseProcessor(marshaler, unmarshaler, engine),
		NewTurnBaseEngine(),
	}
}

func (p *Processor) ProcessNewGame(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	p.engine.NewGame(s)
	listPlayerId := make([]string, 0)
	// deal player
	for _, presence := range s.GetPlayingPresences() {
		if s.IsBet(presence.GetUserId()) {
			listPlayerId = append(listPlayerId, presence.GetUserId())
			s.AddCards(p.engine.Deal(2), presence.GetUserId(), pb.ShanGameHandN0_SHANGAME_HAND_1ST)
		}
	}
	// deal dealer
	s.AddCards(p.engine.Deal(2), "", pb.ShanGameHandN0_SHANGAME_HAND_1ST)
	p.notifyInitialDealCard(
		ctx, nk, logger, dispatcher, s,
	)
	if p.turnBaseEngine == nil {
		p.turnBaseEngine = NewTurnBaseEngine()
	}
	p.turnBaseEngine.Config(
		listPlayerId,
		[]*Round{
			{
				code:   "bet",
				isGlob: true,
				phases: []*Phase{
					{
						code:     "main",
						duration: time.Second * 10,
					},
				},
			},
			{
				code:   "playing",
				isGlob: false,
				phases: []*Phase{
					{
						code:     "main",
						duration: time.Second * 10,
					},
				},
			},
		},
	)

	p.turnBaseEngine.SetCurrentRound("bet")
	p.turnBaseEngine.SetCurrentPlayer(listPlayerId[0])
}

func (p *Processor) ProcessFinishGame(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	updateFinish := s.GetUpdateFinish()
	balanceResult := p.calcRewardForUserPlaying(
		ctx, nk, logger, db, dispatcher, s, updateFinish,
	)
	s.SetBalanceResult(balanceResult)
	p.updateChipByResultGameFinish(ctx, nk, logger, balanceResult)
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_FINISH),
		updateFinish, nil, nil, true,
	)
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_WALLET),
		balanceResult, nil, nil, true,
	)
}

func (p *Processor) ProcessTurnbase(ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	var turnInfo *TurnInfo
	if p.turnBaseEngine != nil {
		turnInfo = p.turnBaseEngine.Loop()
	}
	if turnInfo.isNewRound {
		switch turnInfo.roundCode {
		case "bet":
			s.SetAllowBet(true)
			s.SetAllowAction(false)
		case "playing":
			if s.GetDealerHand().First.Type == pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN {
				s.SetUpdateFinish(s.CalcGameFinish())
				return
			}
			s.InitVisited()
			s.SetAllowBet(false)
			s.SetAllowAction(true)
		}
	}
	if turnInfo.isNewTurn && turnInfo.roundCode == "playing" {
		if s.IsAllVisited() {
			if s.IsDealerMustDraw() {
				cards := p.engine.Deal(1)
				s.AddCards(cards, "", pb.ShanGameHandN0_SHANGAME_HAND_1ST)
				p.notifyDealCard(ctx, nk, logger, dispatcher, s, "", pb.ShanGameHandN0_SHANGAME_HAND_1ST)
			} else {
				s.SetUpdateFinish(s.CalcGameFinish())
				return
			}
		}
	}
	if turnInfo.isNewPhase {
		s.SetVisited(turnInfo.userId)
		s.SetCurrentTurn(turnInfo.userId)
		s.SetUpCountDown(time.Duration(turnInfo.countDown * 1e9))
		p.notifyUpdateTurn(ctx, nk, logger, dispatcher, s)
	}
}

func (p *Processor) ProcessMessageFromUser(
	ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	messages []runtime.MatchData,
	s *entity.MatchState,
) {
	for _, message := range messages {
		switch pb.OpCodeRequest(message.GetOpCode()) {
		case pb.OpCodeRequest_OPCODE_REQUEST_BET:
			if !s.IsAllowBet() {
				continue
			}
			bet := &pb.ShanGameBet{}
			if err := p.unmarshaler.Unmarshal(message.GetData(), bet); err != nil {
				logger.
					WithField("module-game", entity.ModuleName).
					WithField("user-id", message.GetUserId()).
					WithField("request-bet", message.GetData()).
					WithField("error", err).
					Error("error-parse-user-bet-request")
				continue
			}
			bet.UserId = message.GetUserId()
			wallet, err := entity.ReadWalletUser(ctx, nk, logger, bet.UserId)
			if err != nil {
				logger.Error("error.read-user-wallet")
				continue
			}
			switch bet.Code {
			// case pb.ShanGameBetCode_SHANGAME_BET_DOUBLE:
			// 	if s.IsCanDoubleBet(bet.UserId, wallet.Chips) {
			// 		chip := s.DoubleBet(bet.UserId)
			// 		p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, chip, pb.ShanGameHandN0_SHANGAME_HAND_1ST)
			// 	}
			// case pb.ShanGameBetCode_SHANGAME_BET_REBET:
			// 	if s.IsCanRebet(bet.UserId, wallet.Chips) {
			// 		chip := s.Rebet(bet.UserId)
			// 		p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, chip, pb.ShanGameHandN0_SHANGAME_HAND_1ST)
			// 	}
			case pb.ShanGameBetCode_SHANGAME_BET_NORMAL:
				if s.IsCanBet(bet.UserId, wallet.Chips, bet) {
					s.AddBet_inUserBets(bet.UserId, wallet.Chips)
					p.notifyUpdateBet(ctx, nk, logger, dispatcher, s, bet.UserId, bet.Chips, pb.ShanGameHandN0_SHANGAME_HAND_1ST)
				}
			}
		case pb.OpCodeRequest_OPCODE_REQUEST_DECLARE_CARDS:
			if s.GetGameState() != pb.GameState_GameStatePlay || s.GetCurrentTurn() == "" {
				continue
			}
			if s.GetCurrentTurn() != message.GetUserId() {
				continue
			}
			action := &pb.ShanGameAction{}
			if err := p.unmarshaler.Unmarshal(message.GetData(), action); err != nil {
				logger.Error("error.parse-action from [%s]", err.Error())
				continue
			} else {
				wallet, err := entity.ReadWalletUser(ctx, nk, logger, action.UserId)
				if err != nil {
					logger.Error("error.read-wallet %v", wallet, err.Error())
					continue
				}
				action.UserId = message.GetUserId()
				switch action.Code {
				case pb.ShanGameActionCode_SHANGAME_ACTION_HIT:
					if s.IsAllowAction() && s.IsCanHit(action.UserId, s.GetCurrentHandN0()) {
						cards := p.engine.Deal(1)
						s.AddCards(cards, action.UserId, s.GetCurrentHandN0())
						p.broadcastMessage(
							logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
							&pb.ShanGameUpdateDeal{
								IsBanker:                 false,
								IsRevealBankerHiddenCard: false,
								UserId:                   action.UserId,
								HandN0:                   s.GetCurrentHandN0(),
								NewCards:                 cards,
								// Hand:                     s.GetPlayerPartOfHand(action.UserId, s.GetCurrentHandN0()),
							}, nil, nil, true,
						)
						// after that hit, player can't hit anymore -> next hand if possible else next turn
						if !s.IsCanHit(action.UserId, s.GetCurrentHandN0()) {
							p.turnBaseEngine.RePhase()
						}
					}
				case pb.ShanGameActionCode_SHANGAME_ACTION_STAY:
					if s.IsAllowAction() && s.GetCurrentHandN0() == pb.ShanGameHandN0_SHANGAME_HAND_1ST && len(s.GetPlayerPartOfHand(action.UserId, pb.ShanGameHandN0_SHANGAME_HAND_2ND).Cards) == 2 {
						s.SetCurrentHandN0(pb.ShanGameHandN0_SHANGAME_HAND_2ND)
						p.turnBaseEngine.RePhase()
					} else {
						p.turnBaseEngine.NextPhase()
					}
				}
			}
		case pb.OpCodeRequest_OPCODE_REQUEST_INFO_TABLE:
			p.broadcastMessage(
				logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
				&pb.ShanGameUpdateDesk{
					IsNewTurn: false,
					// IsInsuranceTurnEnter: s.IsAllowInsurance(),
					InTurn: s.GetCurrentTurn(),
				}, []runtime.Presence{s.GetPresence(message.GetUserId())}, nil, true,
			)
		}
	}
}

// ********************* Private functions *************************

func (p *Processor) notifyUpdateTurn(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) {
	legalActions := &pb.ShanGameLegalActions{
		UserId:  s.GetCurrentTurn(),
		Actions: s.GetLegalActions(),
	}
	msg := &pb.ShanGameUpdateDesk{
		IsInsuranceTurnEnter: false,
		IsNewTurn:            true,
		InTurn:               s.GetCurrentTurn(),
		Hand_N0:              pb.ShanGameHandN0(s.GetCurrentHandN0()),
		IsUpdateBet:          false,
		Actions:              nil,
		IsSplitHand:          false,
	}
	for _, presence := range s.GetPresences() {
		if presence.GetUserId() == s.GetCurrentTurn() {
			msg.Actions = legalActions
		} else {
			msg.Actions = nil
		}
		p.broadcastMessage(
			logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
			msg, []runtime.Presence{presence}, nil, true,
		)
	}
}

func (p *Processor) notifyUpdateBet(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	userId string,
	chip int64,
	pos pb.ShanGameHandN0,
) {
	bet := &pb.ShanGamePlayerBet{
		UserId: userId,
	}
	if pos == pb.ShanGameHandN0_SHANGAME_HAND_UNSPECIFIED {
		bet.Insurance = chip
	} else if pos == pb.ShanGameHandN0_SHANGAME_HAND_1ST {
		bet.First = chip
	}

	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_TABLE),
		&pb.ShanGameUpdateDesk{
			IsNewTurn:           false,
			IsUpdateBet:         true,
			IsUpdateLegalAction: false,
			IsSplitHand:         false,
			Bet:                 bet,
		},
		nil, nil, true,
	)

	wallet, err := entity.ReadWalletUser(ctx, nk, logger, userId)
	if err != nil {
		logger.Error("error.read-wallet [%v]", userId)
	}
	p.updateChipByResultGameFinish(
		ctx, nk, logger, &pb.BalanceResult{
			Updates: []*pb.BalanceUpdate{
				{
					UserId:            userId,
					AmountChipBefore:  wallet.Chips,
					AmountChipAdd:     -chip,
					AmountChipCurrent: wallet.Chips - chip,
				},
			},
		},
	)
}

func (p *Processor) updateChipByResultGameFinish(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	balanceResult *pb.BalanceResult,
) {
	walletUpdates := make([]*runtime.WalletUpdate, 0, len(balanceResult.Updates))
	for _, update := range balanceResult.Updates {
		amountChip := update.AmountChipCurrent - update.AmountChipBefore
		changeset := map[string]int64{
			"chips": amountChip,
		}
		metadata := map[string]any{"game_reward": entity.ModuleName}
		walletUpdates = append(walletUpdates, &runtime.WalletUpdate{
			UserID:    update.UserId,
			Changeset: changeset,
			Metadata:  metadata,
		})
	}
	if _, err := nk.WalletsUpdate(ctx, walletUpdates, true); err != nil {
		payload, _ := json.Marshal(walletUpdates)
		logger.WithField("payload", string(payload)).
			WithField("err", err).
			Error("wallet-update-error")
	}
}

func (p *Processor) calcRewardForUserPlaying(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	db *sql.DB,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	updateFinish *pb.ShanGameUpdateFinish,
) *pb.BalanceResult {
	listUserPlaying := s.GetPlayingPresences()
	listUserId := make([]string, 0)
	mapUserIdCalcReward := make(map[string]bool, 0)
	for _, u := range listUserPlaying {
		if s.IsBet(u.GetUserId()) {
			listUserId = append(listUserId, u.GetUserId())
			mapUserIdCalcReward[u.GetUserId()] = false
		}
	}
	mapUserWallet := make(map[string]entity.Wallet)
	wallets, err := entity.ReadWalletUsers(
		ctx, nk, logger, listUserId...,
	)
	if err != nil {
		data, _ := p.marshaler.Marshal(updateFinish)
		logger.
			WithField("users", strings.Join(listUserId, ", ")).
			WithField("data", string(data)).
			WithField("err", err).
			Error("error.read-wallet")
		return nil
	}
	for _, w := range wallets {
		mapUserWallet[w.UserId] = w
	}
	balanceResult := pb.BalanceResult{}
	for _, betResult := range updateFinish.BetResults {
		balance := &pb.BalanceUpdate{
			UserId:           betResult.UserId,
			AmountChipBefore: mapUserWallet[betResult.UserId].Chips,
		}
		balance.AmountChipAdd = betResult.First.Total
		if balance.AmountChipAdd > 0 {
			fee := int64(0)
			// presence, ok := s.GetPresence(betResult.UserId).(entity.MyPrecense)

			// viết 1 hàm kiểm tra presence có hay ko thôi
			percentFeeGame := int64(0)
			if s.IsExitsPresence(betResult.UserId) {
				percentFeeGame = s.GetTiLeTienHo_User(betResult.UserId)
			}
			// fee = balance.AmountChipAdd / 100 * int64(percentFeeGame)
			fee = int64((float64(percentFeeGame) / 100) * float64(balance.AmountChipAdd))

			balance.AmountChipCurrent = balance.AmountChipBefore + balance.AmountChipAdd - fee
		} else {
			balance.AmountChipCurrent = balance.AmountChipBefore
		}
		mapUserIdCalcReward[betResult.UserId] = true
		balanceResult.Updates = append(balanceResult.Updates, balance)
	}
	for uid, isChange := range mapUserIdCalcReward {
		if isChange {
			continue
		}
		wallet := mapUserWallet[uid]
		balanceResult.Updates = append(balanceResult.Updates, &pb.BalanceUpdate{
			UserId:            uid,
			AmountChipBefore:  wallet.Chips,
			AmountChipCurrent: wallet.Chips,
			AmountChipAdd:     0,
		})
	}
	return &balanceResult
}

func (p *Processor) notifyInitialDealCard(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
) error {
	// duyệt ds playing đang trong game
	for _, presence := range s.GetPlayingPresences() {
		if s.IsBet(presence.GetUserId()) { // kiểm tra user đã đặt cược hay chưa ?
			p.broadcastMessage( // từ process đưa ra thông báo và thiết lập các thông tin cho broadcastMessage
				logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
				&pb.ShanGameUpdateDeal{
					IsBanker:                 false,
					IsRevealBankerHiddenCard: false,
					UserId:                   presence.GetUserId(),
					NewCards:                 s.GetPlayerHand(presence.GetUserId()).First.Cards,
					Hand:                     s.GetPlayerHand(presence.GetUserId()),
					HandN0:                   pb.ShanGameHandN0_SHANGAME_HAND_1ST,
				}, nil, nil, true,
			)
		}
	}
	dealerCards := []*pb.Card{
		s.GetDealerHand().First.GetCards()[0],
		{
			Rank: pb.CardRank_RANK_UNSPECIFIED,
			Suit: pb.CardSuit_SUIT_UNSPECIFIED,
		},
	}
	p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
		&pb.ShanGameUpdateDeal{
			IsBanker:                 true,
			IsRevealBankerHiddenCard: false,
			UserId:                   "",
			NewCards:                 dealerCards,
			HandN0:                   pb.ShanGameHandN0_SHANGAME_HAND_1ST,
			Hand: &pb.ShanGamePlayerHand{
				First: s.GetDealerHand().First,
			},
		}, nil, nil, true,
	)
	return nil
}

func (p *Processor) notifyDealCard(
	ctx context.Context,
	nk runtime.NakamaModule,
	logger runtime.Logger,
	dispatcher runtime.MatchDispatcher,
	s *entity.MatchState,
	userId string,
	handN0 pb.ShanGameHandN0,
) error {
	isBanker := false
	var hands *pb.ShanGamePlayerHand
	if userId == "" {
		isBanker = true
		hands = s.GetDealerHand()
	} else {
		hands = s.GetPlayerHand(userId)
	}
	var hand *pb.ShanGameHand
	if handN0 == pb.ShanGameHandN0_SHANGAME_HAND_1ST {
		hand = hands.First
	}
	msg := &pb.ShanGameUpdateDeal{
		IsBanker:                 isBanker,
		IsRevealBankerHiddenCard: false,
		HandN0:                   handN0,
		NewCards: []*pb.Card{
			hand.Cards[len(hand.Cards)-1],
		},
		// Hand: hand,
	}
	return p.broadcastMessage(
		logger, dispatcher, int64(pb.OpCodeUpdate_OPCODE_UPDATE_DEAL),
		msg,
		nil, nil, true,
	)
}
