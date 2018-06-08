package main

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	slice "github.com/bradfitz/slice"
	pt "github.com/tchap/go-patricia/patricia"
)

// QuestionResponse represents a response to a question
type QuestionResponse int

// The set of currently supported question responses
const (
	ResponseNone QuestionResponse = iota
	ResponseYes
	ResponseNo
)

// ActionType represents an action to be performed
type ActionType int

// The set of actions possible supported by grv
const (
	ActionNone ActionType = iota
	ActionExit
	ActionSuspend
	ActionRunCommand
	ActionPrompt
	ActionSearchPrompt
	ActionReverseSearchPrompt
	ActionFilterPrompt
	ActionQuestionPrompt
	ActionBranchNamePrompt
	ActionSearch
	ActionReverseSearch
	ActionSearchFindNext
	ActionSearchFindPrev
	ActionClearSearch
	ActionShowStatus
	ActionNextLine
	ActionPrevLine
	ActionNextPage
	ActionPrevPage
	ActionNextHalfPage
	ActionPrevHalfPage
	ActionScrollRight
	ActionScrollLeft
	ActionFirstLine
	ActionLastLine
	ActionSelect
	ActionNextView
	ActionPrevView
	ActionFullScreenView
	ActionToggleViewLayout
	ActionNextTab
	ActionPrevTab
	ActionRemoveView
	ActionAddFilter
	ActionRemoveFilter
	ActionCenterView
	ActionScrollCursorTop
	ActionScrollCursorBottom
	ActionCursorTopView
	ActionCursorMiddleView
	ActionCursorBottomView
	ActionNewTab
	ActionRemoveTab
	ActionAddView
	ActionSplitView
	ActionMouseSelect
	ActionMouseScrollDown
	ActionMouseScrollUp
	ActionCheckoutRef
	ActionCheckoutCommit
	ActionCreateBranch
	ActionCreateContextMenu
	ActionCreateCommandOutputView
	ActionShowAvailableActions
	ActionStageFile
	ActionUnstageFile
	ActionCommit
	ActionShowHelpView
)

// ActionCategory defines the type of an action
type ActionCategory int

// The set of ActionCategory values
const (
	ActionCategoryNone ActionCategory = iota
	ActionCategoryMovement
	ActionCategorySearch
	ActionCategoryViewNavigation
	ActionCategoryGeneral
	ActionCategoryViewSpecific
)

// ActionDescriptor describes an action
type ActionDescriptor struct {
	actionKey      string
	actionCategory ActionCategory
	promptAction   bool
	description    string
	keyBindings    map[ViewID][]string
}

var actionDescriptors = map[ActionType]ActionDescriptor{
	ActionNone: ActionDescriptor{
		description: "Perform no action (NOP)",
	},
	ActionExit: ActionDescriptor{
		actionKey:      "<grv-exit>",
		actionCategory: ActionCategoryGeneral,
		description:    "Exit GRV",
	},
	ActionSuspend: ActionDescriptor{
		actionKey:      "<grv-suspend>",
		actionCategory: ActionCategoryGeneral,
		description:    "Suspend GRV",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-z>"},
		},
	},
	ActionRunCommand: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Run a shell command",
	},
	ActionPrompt: ActionDescriptor{
		actionKey:      "<grv-prompt>",
		actionCategory: ActionCategoryGeneral,
		promptAction:   true,
		description:    "GRV Command prompt",
		keyBindings: map[ViewID][]string{
			ViewMain: {PromptText},
		},
	},
	ActionSearchPrompt: ActionDescriptor{
		actionKey:      "<grv-search-prompt>",
		actionCategory: ActionCategorySearch,
		promptAction:   true,
		description:    "Search forwards",
		keyBindings: map[ViewID][]string{
			ViewMain: {SearchPromptText},
		},
	},
	ActionReverseSearchPrompt: ActionDescriptor{
		actionKey:      "<grv-reverse-search-prompt>",
		actionCategory: ActionCategorySearch,
		promptAction:   true,
		description:    "Search backwards",
		keyBindings: map[ViewID][]string{
			ViewMain: {ReverseSearchPromptText},
		},
	},
	ActionFilterPrompt: ActionDescriptor{
		actionKey:      "<grv-filter-prompt>",
		actionCategory: ActionCategoryViewSpecific,
		promptAction:   true,
		description:    "Add filter",
		keyBindings: map[ViewID][]string{
			ViewCommit: {"<C-q>"},
			ViewRef:    {"<C-q>"},
		},
	},
	ActionQuestionPrompt: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		promptAction:   true,
		description:    "Prompt the user with a question",
	},
	ActionBranchNamePrompt: ActionDescriptor{
		actionKey:      "<grv-branch-name-prompt>",
		actionCategory: ActionCategoryGeneral,
		promptAction:   true,
		description:    "Create a new branch",
		keyBindings: map[ViewID][]string{
			ViewRef:    {"b"},
			ViewCommit: {"b"},
		},
	},
	ActionSearch: ActionDescriptor{
		actionCategory: ActionCategorySearch,
		description:    "Perform search forwards",
	},
	ActionReverseSearch: ActionDescriptor{
		actionCategory: ActionCategorySearch,
		description:    "Perform search backwards",
	},
	ActionSearchFindNext: ActionDescriptor{
		actionKey:      "<grv-search-find-next>",
		actionCategory: ActionCategorySearch,
		description:    "Move to next search match",
		keyBindings: map[ViewID][]string{
			ViewAll: {"n"},
		},
	},
	ActionSearchFindPrev: ActionDescriptor{
		actionKey:      "<grv-search-find-prev>",
		actionCategory: ActionCategorySearch,
		description:    "Move to previous search match",
		keyBindings: map[ViewID][]string{
			ViewAll: {"N"},
		},
	},
	ActionClearSearch: ActionDescriptor{
		actionKey:      "<grv-clear-search>",
		actionCategory: ActionCategorySearch,
		description:    "Clear search",
	},
	ActionShowStatus: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Display message in status bar",
	},
	ActionNextLine: ActionDescriptor{
		actionKey:      "<grv-next-line>",
		actionCategory: ActionCategoryMovement,
		description:    "Move down one line",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<Down>", "j"},
		},
	},
	ActionPrevLine: ActionDescriptor{
		actionKey:      "<grv-prev-line>",
		actionCategory: ActionCategoryMovement,
		description:    "Move up one line",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<Up>", "k"},
		},
	},
	ActionNextPage: ActionDescriptor{
		actionKey:      "<grv-next-page>",
		actionCategory: ActionCategoryMovement,
		description:    "Move one page down",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<PageDown>", "<C-f>"},
		},
	},
	ActionPrevPage: ActionDescriptor{
		actionKey:      "<grv-prev-page>",
		actionCategory: ActionCategoryMovement,
		description:    "Move one page up",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<PageUp>", "<C-b>"},
		},
	},
	ActionNextHalfPage: ActionDescriptor{
		actionKey:      "<grv-next-half-page>",
		actionCategory: ActionCategoryMovement,
		description:    "Move half page down",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-d>"},
		},
	},
	ActionPrevHalfPage: ActionDescriptor{
		actionKey:      "<grv-prev-half-page>",
		actionCategory: ActionCategoryMovement,
		description:    "Move half page up",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-u>"},
		},
	},
	ActionScrollRight: ActionDescriptor{
		actionKey:      "<grv-scroll-right>",
		actionCategory: ActionCategoryMovement,
		description:    "Scroll right",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<Right>", "l"},
		},
	},
	ActionScrollLeft: ActionDescriptor{
		actionKey:      "<grv-scroll-left>",
		actionCategory: ActionCategoryMovement,
		description:    "Scroll left",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<Left>", "h"},
		},
	},
	ActionFirstLine: ActionDescriptor{
		actionKey:      "<grv-first-line>",
		actionCategory: ActionCategoryMovement,
		description:    "Move to first line",
		keyBindings: map[ViewID][]string{
			ViewAll: {"gg"},
		},
	},
	ActionLastLine: ActionDescriptor{
		actionKey:      "<grv-last-line>",
		actionCategory: ActionCategoryMovement,
		description:    "Move to last line",
		keyBindings: map[ViewID][]string{
			ViewAll: {"G"},
		},
	},
	ActionSelect: ActionDescriptor{
		actionKey:      "<grv-select>",
		actionCategory: ActionCategoryGeneral,
		description:    "Select item (opens listener view if none exists)",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<Enter>"},
		},
	},
	ActionNextView: ActionDescriptor{
		actionKey:      "<grv-next-view>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Move to next view",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-w>w", "<C-w><C-w>", "<Tab>"},
		},
	},
	ActionPrevView: ActionDescriptor{
		actionKey:      "<grv-prev-view>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Move to previous view",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-w>W", "<S-Tab>"},
		},
	},
	ActionFullScreenView: ActionDescriptor{
		actionKey:      "<grv-full-screen-view>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Toggle current view full screen",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-w>o", "<C-w><C-o>", "f"},
		},
	},
	ActionToggleViewLayout: ActionDescriptor{
		actionKey:      "<grv-toggle-view-layout>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Toggle view layout",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-w>t"},
		},
	},
	ActionNextTab: ActionDescriptor{
		actionKey:      "<grv-next-tab>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Move to next tab",
		keyBindings: map[ViewID][]string{
			ViewAll: {"gt"},
		},
	},
	ActionPrevTab: ActionDescriptor{
		actionKey:      "<grv-prev-tab>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Move to previous tab",
		keyBindings: map[ViewID][]string{
			ViewAll: {"gT"},
		},
	},
	ActionRemoveView: ActionDescriptor{
		actionKey:      "<grv-remove-view>",
		actionCategory: ActionCategoryViewNavigation,
		description:    "Close view (or close tab if empty)",
		keyBindings: map[ViewID][]string{
			ViewAll: {"q"},
		},
	},
	ActionAddFilter: ActionDescriptor{
		actionCategory: ActionCategoryViewSpecific,
		description:    "Add filter",
	},
	ActionRemoveFilter: ActionDescriptor{
		actionKey:      "<grv-remove-filter>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Remove filter",
		keyBindings: map[ViewID][]string{
			ViewCommit: {"<C-r>"},
			ViewRef:    {"<C-r>"},
		},
	},
	ActionCenterView: ActionDescriptor{
		actionKey:      "<grv-center-view>",
		actionCategory: ActionCategoryMovement,
		description:    "Center view",
		keyBindings: map[ViewID][]string{
			ViewAll: {"z.", "zz"},
		},
	},
	ActionScrollCursorTop: ActionDescriptor{
		actionKey:      "<grv-scroll-cursor-top>",
		actionCategory: ActionCategoryMovement,
		description:    "Scroll the screen so cursor is at the top",
		keyBindings: map[ViewID][]string{
			ViewAll: {"zt"},
		},
	},
	ActionScrollCursorBottom: ActionDescriptor{
		actionKey:      "<grv-scroll-cursor-bottom>",
		actionCategory: ActionCategoryMovement,
		description:    "Scroll the screen so cursor is at the bottom",
		keyBindings: map[ViewID][]string{
			ViewAll: {"zb"},
		},
	},
	ActionCursorTopView: ActionDescriptor{
		actionKey:      "<grv-cursor-top-view>",
		actionCategory: ActionCategoryMovement,
		description:    "Move to the first line of the page",
		keyBindings: map[ViewID][]string{
			ViewAll: {"H"},
		},
	},
	ActionCursorMiddleView: ActionDescriptor{
		actionKey:      "<grv-cursor-middle-view>",
		actionCategory: ActionCategoryMovement,
		description:    "Move to the middle line of the page",
		keyBindings: map[ViewID][]string{
			ViewAll: {"M"},
		},
	},
	ActionCursorBottomView: ActionDescriptor{
		actionKey:      "<grv-cursor-bottom-view>",
		actionCategory: ActionCategoryMovement,
		description:    "Move to the last line of the page",
		keyBindings: map[ViewID][]string{
			ViewAll: {"L"},
		},
	},
	ActionNewTab: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Add a new tab",
	},
	ActionRemoveTab: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Remove the active tab",
	},
	ActionAddView: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Add a new view",
	},
	ActionSplitView: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Split the current view with a new view",
	},
	ActionMouseSelect: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Mouse select",
	},
	ActionMouseScrollDown: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Mouse scroll down",
	},
	ActionMouseScrollUp: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Mouse scroll up",
	},
	ActionCheckoutRef: ActionDescriptor{
		actionKey:      "<grv-checkout-ref>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Checkout ref",
		keyBindings: map[ViewID][]string{
			ViewRef: {"c"},
		},
	},
	ActionCheckoutCommit: ActionDescriptor{
		actionKey:      "<grv-checkout-commit>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Checkout commit",
		keyBindings: map[ViewID][]string{
			ViewCommit: {"c"},
		},
	},
	ActionCreateBranch: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Create a branch",
	},
	ActionCreateContextMenu: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Create a context menu",
	},
	ActionCreateCommandOutputView: ActionDescriptor{
		actionCategory: ActionCategoryGeneral,
		description:    "Create a command output view",
	},
	ActionShowAvailableActions: ActionDescriptor{
		actionKey:      "<grv-show-available-actions>",
		actionCategory: ActionCategoryGeneral,
		description:    "Show available actions for the selected row",
		keyBindings: map[ViewID][]string{
			ViewAll: {"<C-a>"},
		},
	},
	ActionStageFile: ActionDescriptor{
		actionKey:      "<grv-stage-file>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Stage",
		keyBindings: map[ViewID][]string{
			ViewGitStatus: {"a"},
		},
	},
	ActionUnstageFile: ActionDescriptor{
		actionKey:      "<grv-unstage-file>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Unstage",
		keyBindings: map[ViewID][]string{
			ViewGitStatus: {"u"},
		},
	},
	ActionCommit: ActionDescriptor{
		actionKey:      "<grv-action-commit>",
		actionCategory: ActionCategoryViewSpecific,
		description:    "Commit",
		keyBindings: map[ViewID][]string{
			ViewGitStatus: {"c"},
		},
	},
	ActionShowHelpView: ActionDescriptor{
		actionKey:      "<grv-show-help>",
		actionCategory: ActionCategoryGeneral,
		description:    "Show the help view",
	},
}

var whitespaceBindingRegex = regexp.MustCompile(`^(.*\s+.*)+$`)

var actionKeys = map[string]ActionType{}

func init() {
	for actionType, actionDescriptor := range actionDescriptors {
		if actionDescriptor.actionKey != "" {
			actionKeys[actionDescriptor.actionKey] = actionType
		}
	}
}

// Action represents a type of actions and its arguments to be executed
type Action struct {
	ActionType ActionType
	Args       []interface{}
}

// CreateViewArgs contains the fields required to create and configure a view
type CreateViewArgs struct {
	viewID               ViewID
	viewArgs             []interface{}
	registerViewListener RegisterViewListener
}

// ActionAddViewArgs contains arguments the ActionAddView action requires
type ActionAddViewArgs struct {
	CreateViewArgs
}

// ActionSplitViewArgs contains arguments the ActionSplitView action requires
type ActionSplitViewArgs struct {
	CreateViewArgs
	orientation ContainerOrientation
}

// ActionPromptArgs contains arguments to an action that displays a prompt
type ActionPromptArgs struct {
	keys       string
	terminated bool
}

// ActionQuestionPromptArgs contains arguments to configure a question prompt
type ActionQuestionPromptArgs struct {
	question      string
	answers       []string
	defaultAnswer string
	onAnswer      func(string)
}

// ActionCreateContextMenuArgs contains arguments to create and configure a context menu
type ActionCreateContextMenuArgs struct {
	config        ContextMenuConfig
	viewDimension ViewDimension
}

// ActionCreateCommandOutputViewArgs contains arguments to create and configure a command output view
type ActionCreateCommandOutputViewArgs struct {
	command       string
	viewDimension ViewDimension
	onCreation    func(commandOutputProcessor CommandOutputProcessor)
}

// ActionRunCommandArgs contains arguments to run a command and process
// the status and output
type ActionRunCommandArgs struct {
	command        string
	interactive    bool
	promptForInput bool
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	beforeStart    func(cmd *exec.Cmd)
	onStart        func(cmd *exec.Cmd)
	onComplete     func(err error, exitStatus int) error
}

// ViewHierarchy is a list of views parent to child
type ViewHierarchy []ViewID

// BindingType specifies the type a key sequence is bound to
type BindingType int

// The types a key sequence can by bound to
const (
	BtAction BindingType = iota
	BtKeystring
)

// Binding is the entity a key sequence is bound to
// This is either an action or a key sequence
type Binding struct {
	bindingType BindingType
	actionType  ActionType
	keystring   string
}

func newActionBinding(actionType ActionType) Binding {
	return Binding{
		bindingType: BtAction,
		actionType:  actionType,
	}
}

func newKeystringBinding(keystring string) Binding {
	return Binding{
		bindingType: BtKeystring,
		keystring:   keystring,
		actionType:  ActionNone,
	}
}

// KeyBindings exposes key bindings that have been configured and allows new bindings to be set
type KeyBindings interface {
	Binding(viewHierarchy ViewHierarchy, keystring string) (binding Binding, isPrefix bool)
	SetActionBinding(viewID ViewID, keystring string, actionType ActionType)
	SetKeystringBinding(viewID ViewID, keystring, mappedKeystring string)
	RemoveBinding(viewID ViewID, keystring string) (removed bool)
	KeyStrings(actionType ActionType, viewID ViewID) (keystrings []BoundKeyString)
	GenerateHelpSections(Config) []*HelpSection
}

// BoundKeyString is a keystring bound to an action
type BoundKeyString struct {
	keystring          string
	userDefinedBinding bool
}

// KeyBindingManager manages key bindings in grv
type KeyBindingManager struct {
	bindings           map[ViewID]*pt.Trie
	helpFormat         map[ActionType]map[ViewID][]BoundKeyString
	userDefinedBinding bool
}

// NewKeyBindingManager creates a new instance
func NewKeyBindingManager() KeyBindings {
	keyBindingManager := &KeyBindingManager{
		bindings:   make(map[ViewID]*pt.Trie),
		helpFormat: make(map[ActionType]map[ViewID][]BoundKeyString),
	}

	keyBindingManager.setDefaultKeyBindings()
	keyBindingManager.userDefinedBinding = true

	return keyBindingManager
}

// Binding returns the Binding bound to the provided key sequence for the view hierarchy provided
// If no binding exists or the provided key sequence is a prefix to a binding then an action binding with action ActionNone is returned and a boolean indicating whether there is a prefix match
func (keyBindingManager *KeyBindingManager) Binding(viewHierarchy ViewHierarchy, keystring string) (Binding, bool) {
	viewHierarchy = append(viewHierarchy, ViewAll)
	isPrefix := false

	for _, viewID := range viewHierarchy {
		if viewBindings, ok := keyBindingManager.bindings[viewID]; ok {
			if binding := viewBindings.Get(pt.Prefix(keystring)); binding != nil {
				return binding.(Binding), false
			} else if viewBindings.MatchSubtree(pt.Prefix(keystring)) {
				isPrefix = true
			}
		}
	}

	return newActionBinding(ActionNone), isPrefix
}

// SetActionBinding allows an action to be bound to the provided key sequence and view
func (keyBindingManager *KeyBindingManager) SetActionBinding(viewID ViewID, keystring string, actionType ActionType) {
	viewBindings := keyBindingManager.getOrCreateViewBindings(viewID)
	viewBindings.Set(pt.Prefix(keystring), newActionBinding(actionType))
	keyBindingManager.updateHelpFormat(actionType, viewID, keystring)
}

// SetKeystringBinding allows a key sequence to be bound to the provided key sequence and view
func (keyBindingManager *KeyBindingManager) SetKeystringBinding(viewID ViewID, keystring, mappedKeystring string) {
	keyBindingManager.RemoveBinding(viewID, keystring)

	viewBindings := keyBindingManager.getOrCreateViewBindings(viewID)
	viewBindings.Set(pt.Prefix(keystring), newKeystringBinding(mappedKeystring))

	if actionType, ok := actionKeys[mappedKeystring]; ok {
		keyBindingManager.updateHelpFormat(actionType, viewID, keystring)
	}
}

func (keyBindingManager *KeyBindingManager) getOrCreateViewBindings(viewID ViewID) *pt.Trie {
	viewBindings, ok := keyBindingManager.bindings[viewID]
	if ok {
		return viewBindings
	}

	viewBindings = pt.NewTrie()
	keyBindingManager.bindings[viewID] = viewBindings
	return viewBindings
}

func (keyBindingManager *KeyBindingManager) updateHelpFormat(actionType ActionType, viewID ViewID, keystring string) {
	if strings.HasPrefix(keystring, "<grv-") {
		return
	}

	viewBindings, ok := keyBindingManager.helpFormat[actionType]
	if !ok {
		viewBindings = map[ViewID][]BoundKeyString{}
		keyBindingManager.helpFormat[actionType] = viewBindings
	}

	keystrings, ok := viewBindings[viewID]
	if !ok {
		keystrings = []BoundKeyString{}
	}

	viewBindings[viewID] = append(keystrings, BoundKeyString{
		keystring:          keystring,
		userDefinedBinding: keyBindingManager.userDefinedBinding,
	})
}

// RemoveBinding removes the binding for the provided keystring if it exists
func (keyBindingManager *KeyBindingManager) RemoveBinding(viewID ViewID, keystring string) (removed bool) {
	binding, _ := keyBindingManager.Binding([]ViewID{viewID}, keystring)

	if viewBindings, ok := keyBindingManager.bindings[viewID]; ok {
		removed = viewBindings.Delete(pt.Prefix(keystring))
	}

	if binding.actionType != ActionNone || binding.keystring != "" {
		keyBindingManager.removeHelpFormatEntry(binding, viewID, keystring)
	}

	return
}

func (keyBindingManager *KeyBindingManager) removeHelpFormatEntry(binding Binding, viewID ViewID, keystring string) {
	var actionType ActionType

	if binding.bindingType == BtAction {
		actionType = binding.actionType
	} else if binding.bindingType == BtKeystring {
		if mappedActionType, ok := actionKeys[binding.keystring]; ok {
			actionType = mappedActionType
		}
	}

	viewBindings, ok := keyBindingManager.helpFormat[actionType]
	if !ok {
		return
	}

	keystrings, ok := viewBindings[viewID]
	if !ok {
		return
	}

	updatedKeystrings := []BoundKeyString{}

	for _, key := range keystrings {
		if key.keystring != keystring {
			updatedKeystrings = append(updatedKeystrings, key)
		}
	}

	viewBindings[viewID] = updatedKeystrings
}

// KeyStrings returns the keystrings bound to the provided action and view
func (keyBindingManager *KeyBindingManager) KeyStrings(actionType ActionType, viewID ViewID) (keystrings []BoundKeyString) {
	viewBindings, ok := keyBindingManager.helpFormat[actionType]
	if !ok {
		return
	}

	keystrings, _ = viewBindings[viewID]
	return
}

func (keyBindingManager *KeyBindingManager) setDefaultKeyBindings() {
	for actionKey, actionType := range actionKeys {
		keyBindingManager.SetActionBinding(ViewAll, actionKey, actionType)
	}

	for actionType, actionDescriptor := range actionDescriptors {
		for viewID, keys := range actionDescriptor.keyBindings {
			for _, key := range keys {
				keyBindingManager.SetActionBinding(viewID, key, actionType)
			}
		}
	}
}

// GenerateHelpSections generates key binding help sections
func (keyBindingManager *KeyBindingManager) GenerateHelpSections(config Config) []*HelpSection {
	helpSections := []*HelpSection{
		&HelpSection{
			title: HelpSectionText{text: "Key Bindings"},
			description: []HelpSectionText{
				HelpSectionText{text: "The following tables contain default and user configured key bindings"},
			},
		},
	}

	type KeyBindingSection struct {
		title        string
		actionFilter actionFilter
	}

	keyBindingSections := []KeyBindingSection{
		KeyBindingSection{
			title: "Movement",
			actionFilter: func(actionDescriptor ActionDescriptor) bool {
				return actionDescriptor.actionCategory == ActionCategoryMovement
			},
		},
		KeyBindingSection{
			title: "Search",
			actionFilter: func(actionDescriptor ActionDescriptor) bool {
				return actionDescriptor.actionCategory == ActionCategorySearch
			},
		},
		KeyBindingSection{
			title: "View Navigation",
			actionFilter: func(actionDescriptor ActionDescriptor) bool {
				return actionDescriptor.actionCategory == ActionCategoryViewNavigation
			},
		},
		KeyBindingSection{
			title: "General",
			actionFilter: func(actionDescriptor ActionDescriptor) bool {
				return actionDescriptor.actionCategory == ActionCategoryGeneral
			},
		},
	}

	for _, KeyBindingSection := range keyBindingSections {
		helpSections = append(helpSections, &HelpSection{
			description: []HelpSectionText{
				HelpSectionText{text: KeyBindingSection.title, themeComponentID: CmpHelpViewSectionSubTitle},
			},
			tableFormatter: keyBindingManager.generateKeyBindingsTable(config, KeyBindingSection.actionFilter),
		})
	}

	return helpSections
}

type actionFilter func(ActionDescriptor) bool

func (keyBindingManager *KeyBindingManager) generateKeyBindingsTable(config Config, filter actionFilter) *TableFormatter {
	headers := []TableHeader{
		TableHeader{text: "Key Bindings", themeComponentID: CmpHelpViewSectionTableHeader},
		TableHeader{text: "Action", themeComponentID: CmpHelpViewSectionTableHeader},
		TableHeader{text: "Description", themeComponentID: CmpHelpViewSectionTableHeader},
	}

	tableFormatter := NewTableFormatterWithHeaders(headers, config)
	tableFormatter.SetGridLines(true)

	type matchingActionDescriptor struct {
		actionType       ActionType
		actionDescriptor ActionDescriptor
	}

	matchingActionDescriptors := []matchingActionDescriptor{}

	for actionType, actionDescriptor := range actionDescriptors {
		if actionDescriptor.actionKey != "" && filter(actionDescriptor) {
			matchingActionDescriptors = append(matchingActionDescriptors, matchingActionDescriptor{
				actionType:       actionType,
				actionDescriptor: actionDescriptor,
			})
		}
	}

	slice.Sort(matchingActionDescriptors, func(i, j int) bool {
		return matchingActionDescriptors[i].actionDescriptor.actionKey < matchingActionDescriptors[j].actionDescriptor.actionKey
	})

	tableFormatter.Resize(uint(len(matchingActionDescriptors)))

	for rowIndex, matchingActionDescriptor := range matchingActionDescriptors {
		seenKeyBindings := map[string]bool{}
		keyBindings := []BoundKeyString{}

		viewIDs := []ViewID{}
		if len(matchingActionDescriptor.actionDescriptor.keyBindings) == 0 {
			viewIDs = append(viewIDs, ViewAll)
		} else {
			for viewID := range matchingActionDescriptor.actionDescriptor.keyBindings {
				viewIDs = append(viewIDs, viewID)
			}
		}

		for _, viewID := range viewIDs {
			for _, keyBinding := range keyBindingManager.KeyStrings(matchingActionDescriptor.actionType, viewID) {
				if _, exists := seenKeyBindings[keyBinding.keystring]; !exists {
					keyBindings = append(keyBindings, keyBinding)
					seenKeyBindings[keyBinding.keystring] = true
				}
			}
		}

		if len(keyBindings) == 0 {
			tableFormatter.SetCellWithStyle(uint(rowIndex), 0, CmpHelpViewSectionTableCellSeparator, "%v", "None")
		} else {
			for bindingIndex, keyBinding := range keyBindings {
				themeComponentID := CmpHelpViewSectionTableRow
				if keyBinding.userDefinedBinding {
					themeComponentID = CmpHelpViewSectionTableRowHighlighted
				}

				keystringContainsWhitespace := whitespaceBindingRegex.MatchString(keyBinding.keystring)

				if keystringContainsWhitespace {
					tableFormatter.AppendToCellWithStyle(uint(rowIndex), 0, themeComponentID, `"`)
				}

				tableFormatter.AppendToCellWithStyle(uint(rowIndex), 0, themeComponentID, "%v", keyBinding.keystring)

				if keystringContainsWhitespace {
					tableFormatter.AppendToCellWithStyle(uint(rowIndex), 0, themeComponentID, `"`)
				}

				if bindingIndex != len(keyBindings)-1 {
					tableFormatter.AppendToCellWithStyle(uint(rowIndex), 0, CmpHelpViewSectionTableCellSeparator, "%v", ", ")
				}
			}
		}

		tableFormatter.SetCellWithStyle(uint(rowIndex), 1, CmpHelpViewSectionTableRow, "%v", matchingActionDescriptor.actionDescriptor.actionKey)
		tableFormatter.SetCellWithStyle(uint(rowIndex), 2, CmpHelpViewSectionTableRow, "%v", matchingActionDescriptor.actionDescriptor.description)
	}

	return tableFormatter
}

func isValidAction(action string) bool {
	_, valid := actionKeys[action]
	return valid
}

// IsPromptAction returns true if the action presents a prompt
func IsPromptAction(actionType ActionType) bool {
	if actionDescriptor, exists := actionDescriptors[actionType]; exists {
		return actionDescriptor.promptAction
	}

	return false
}

// MouseEventAction maps a mouse event to an action
func MouseEventAction(mouseEvent MouseEvent) (action Action, err error) {
	switch mouseEvent.mouseEventType {
	case MetLeftClick:
		action = Action{
			ActionType: ActionMouseSelect,
			Args:       []interface{}{mouseEvent},
		}
	case MetScrollDown:
		action = Action{ActionType: ActionMouseScrollDown}
	case MetScrollUp:
		action = Action{ActionType: ActionMouseScrollUp}
	default:
		err = fmt.Errorf("Unknown MouseEventType %v", mouseEvent.mouseEventType)
	}

	return
}

// GetMouseEventFromAction converts a MouseEvent into an Action that can be processed by a view
func GetMouseEventFromAction(action Action) (mouseEvent MouseEvent, err error) {
	if len(action.Args) == 0 {
		err = fmt.Errorf("Expected MouseEvent arg")
		return
	}

	mouseEvent, ok := action.Args[0].(MouseEvent)
	if !ok {
		err = fmt.Errorf("Expected first argument to have type MouseEvent but has type: %T", action.Args[0])
	}

	return
}

// YesNoQuestion generates an action that will prompt the user for a yes/no response
// The onResponse handler is called when an answer is received
func YesNoQuestion(question string, onResponse func(QuestionResponse)) Action {
	return Action{
		ActionType: ActionQuestionPrompt,
		Args: []interface{}{ActionQuestionPromptArgs{
			question: question,
			answers:  []string{"y", "n"},
			onAnswer: func(answer string) {
				var response QuestionResponse
				switch answer {
				case "y":
					response = ResponseYes
				case "n":
					response = ResponseNo
				default:
					response = ResponseNone
				}

				onResponse(response)
			},
		}},
	}
}
