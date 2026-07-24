import { Component, OnInit, ElementRef, ViewChild, AfterViewChecked, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { Subject, Subscription, interval } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

export interface ChatThread {
  contact: {
    id: string;
    name: string;
    email: string;
    role: string;
  };
  lastMessage: any;
  unreadCount: number;
  messages: any[];
}

@Component({
  selector: 'app-messages',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './messages.component.html',
  styleUrls: ['./messages.component.css']
})
export class MessagesComponent implements OnInit, AfterViewChecked, OnDestroy {
  @ViewChild('chatScrollContainer') private chatScrollContainer!: ElementRef;

  // Sidebar section: 'chats' | 'drafts' | 'trash'
  activeSection: 'chats' | 'drafts' | 'trash' = 'chats';

  isLoading: boolean = false;
  allMessages: any[] = [];
  chatThreads: ChatThread[] = [];
  selectedContact: { id: string; name: string; email: string; role: string } | null = null;
  activeConversationMessages: any[] = [];

  // Pagination & Search in Messages
  inboxMessages: any[] = [];
  sentMessages: any[] = [];
  draftsList: any[] = [];
  trashList: any[] = [];

  inboxPage: number = 1;
  sentPage: number = 1;
  pageSize: number = 20;
  totalInboxCount: number = 0;
  totalSentCount: number = 0;
  hasMoreInbox: boolean = false;
  hasMoreSent: boolean = false;

  messageFilterQuery: string = '';
  private messageSearchSubject = new Subject<string>();

  // Chat Input Dock State & Drafts
  chatInput: string = '';
  chatSubject: string = '';
  showSubjectInput: boolean = false;
  isSending: boolean = false;
  currentDraftId: string | null = null;
  validationError: string | null = null;
  private autoSaveDraftSubject = new Subject<void>();

  // Search & New Chat Modal State
  searchQuery: string = '';
  searchResults: any[] = [];
  isSearching: boolean = false;
  searchError: string | null = null;
  activeTab: 'chats' | 'search' = 'chats';

  // Search Cache & Debouncing
  private recipientSearchSubject = new Subject<string>();
  private searchCache = new Map<string, any[]>();

  // Unread Count
  inboxUnreadCount: number = 0;

  // Toast feedback
  toastMessage: string | null = null;
  toastType: 'success' | 'error' = 'success';

  private shouldScrollToBottom: boolean = false;
  private subs: Subscription[] = [];

  constructor(
    private api: ApiService,
    public auth: AuthService,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.setupSearchDebounce();
    this.setupAutoSaveDraftDebounce();
    this.setupMessageSearchDebounce();
    this.loadData();
    this.checkQueryParams();
    this.refreshUnreadCount();

    const pollSub = interval(8000).subscribe(() => {
      this.refreshQuietly();
    });
    this.subs.push(pollSub);
  }

  ngOnDestroy() {
    this.subs.forEach(s => s.unsubscribe());
  }

  checkQueryParams() {
    this.route.queryParams.subscribe(params => {
      if (params['recipientId'] || params['userId']) {
        const id = params['recipientId'] || params['userId'];
        this.openChatByUserId(id);
      } else if (params['email']) {
        this.openChatByEmail(params['email']);
      }
    });
  }

  openChatByUserId(userId: string) {
    const normId = String(userId || '').toLowerCase();
    const existingThread = this.chatThreads.find(t => String(t.contact.id || '').toLowerCase() === normId);
    if (existingThread) {
      this.selectContact(existingThread.contact);
      return;
    }
    this.api.searchUsers(userId).subscribe({
      next: (results) => {
        const found = (results || []).find(u => String(u.id || u.ID || '').toLowerCase() === normId);
        if (found) {
          this.startChatWithUser(found);
        }
      }
    });
  }

  openChatByEmail(email: string) {
    this.api.getUserByEmail(email).subscribe({
      next: (user) => {
        if (user) {
          this.startChatWithUser(user);
        }
      }
    });
  }

  ngAfterViewChecked() {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  get currentUser(): any {
    return this.auth.getCurrentUser();
  }

  // ── Debounce Setup ──────────────────────────────────────────────────────────

  setupSearchDebounce() {
    const sub = this.recipientSearchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged()
    ).subscribe(q => {
      this.executeRecipientSearch(q);
    });
    this.subs.push(sub);
  }

  setupAutoSaveDraftDebounce() {
    const sub = this.autoSaveDraftSubject.pipe(
      debounceTime(1000)
    ).subscribe(() => {
      this.performAutoSaveDraft();
    });
    this.subs.push(sub);
  }

  setupMessageSearchDebounce() {
    const sub = this.messageSearchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged()
    ).subscribe(() => {
      this.inboxPage = 1;
      this.sentPage = 1;
      this.loadData();
    });
    this.subs.push(sub);
  }

  // ── Data Loading & Pagination ──────────────────────────────────────────────

  loadData() {
    this.isLoading = true;
    this.api.getInboxMessages(this.inboxPage, this.pageSize, this.messageFilterQuery).subscribe({
      next: (inboxRes) => {
        this.inboxMessages = inboxRes.messages || [];
        this.totalInboxCount = inboxRes.total || 0;
        this.hasMoreInbox = this.inboxMessages.length < this.totalInboxCount;

        this.api.getSentMessages(this.sentPage, this.pageSize, this.messageFilterQuery).subscribe({
          next: (sentRes) => {
            this.sentMessages = sentRes.messages || [];
            this.totalSentCount = sentRes.total || 0;
            this.hasMoreSent = this.sentMessages.length < this.totalSentCount;

            this.combineAndGroupMessages(this.inboxMessages, this.sentMessages);
            this.isLoading = false;
          },
          error: (err) => {
            console.error('Failed to load sent messages:', err);
            this.isLoading = false;
          }
        });
      },
      error: (err) => {
        console.error('Failed to load inbox messages:', err);
        this.isLoading = false;
      }
    });

    this.loadDrafts();
    this.loadTrash();
    this.refreshUnreadCount();
  }

  loadMoreInbox() {
    if (!this.hasMoreInbox) return;
    this.inboxPage++;
    this.api.getInboxMessages(this.inboxPage, this.pageSize, this.messageFilterQuery).subscribe({
      next: (res) => {
        const newMsgs = res.messages || [];
        this.inboxMessages = [...this.inboxMessages, ...newMsgs];
        this.totalInboxCount = res.total || 0;
        this.hasMoreInbox = this.inboxMessages.length < this.totalInboxCount;
        this.combineAndGroupMessages(this.inboxMessages, this.sentMessages);
      }
    });
  }

  loadMoreSent() {
    if (!this.hasMoreSent) return;
    this.sentPage++;
    this.api.getSentMessages(this.sentPage, this.pageSize, this.messageFilterQuery).subscribe({
      next: (res) => {
        const newMsgs = res.messages || [];
        this.sentMessages = [...this.sentMessages, ...newMsgs];
        this.totalSentCount = res.total || 0;
        this.hasMoreSent = this.sentMessages.length < this.totalSentCount;
        this.combineAndGroupMessages(this.inboxMessages, this.sentMessages);
      }
    });
  }

  loadDrafts() {
    this.api.getDrafts().subscribe({
      next: (drafts) => {
        this.draftsList = drafts || [];
      }
    });
  }

  loadTrash() {
    this.api.getTrash().subscribe({
      next: (trash) => {
        this.trashList = trash || [];
      }
    });
  }

  refreshQuietly() {
    this.api.getInboxMessages(this.inboxPage, this.pageSize, this.messageFilterQuery).subscribe({
      next: (inboxRes) => {
        this.inboxMessages = inboxRes.messages || [];
        this.totalInboxCount = inboxRes.total || 0;
        this.api.getSentMessages(this.sentPage, this.pageSize, this.messageFilterQuery).subscribe({
          next: (sentRes) => {
            this.sentMessages = sentRes.messages || [];
            this.totalSentCount = sentRes.total || 0;
            this.combineAndGroupMessages(this.inboxMessages, this.sentMessages);
          }
        });
      }
    });
    this.refreshUnreadCount();
  }

  refreshUnreadCount() {
    this.api.getUnreadCount().subscribe({
      next: (res) => {
        this.inboxUnreadCount = res.count || 0;
      }
    });
  }

  onMessageFilterInput() {
    this.messageSearchSubject.next(this.messageFilterQuery);
  }

  normalizeContact(c: any): { id: string; name: string; email: string; role: string } | null {
    if (!c) return null;
    const id = c.id || c.ID || c.recipient_id || c.sender_id;
    if (!id) return null;
    return {
      id: String(id),
      name: c.name || c.Name || c.recipient_name || c.sender_name || 'User',
      email: c.email || c.Email || c.recipient_email || c.sender_email || '',
      role: c.role || c.Role || c.recipient_role || c.sender_role || 'Member'
    };
  }

  combineAndGroupMessages(inbox: any[], sent: any[]) {
    const currentUserId = String(this.currentUser?.id || this.currentUser?.ID || '').toLowerCase();
    this.allMessages = [...inbox, ...sent];

    const threadMap = new Map<string, ChatThread>();

    for (const msg of this.allMessages) {
      const recipientId = String(msg.recipient_id || '').toLowerCase();
      const senderId = String(msg.sender_id || '').toLowerCase();
      const isIncoming = recipientId === currentUserId;
      const otherId = isIncoming ? senderId : recipientId;
      const otherName = isIncoming ? msg.sender_name : msg.recipient_name;
      const otherEmail = isIncoming ? msg.sender_email : msg.recipient_email;
      const otherRole = isIncoming ? msg.sender_role : msg.recipient_role;

      if (!otherId) continue;

      if (!threadMap.has(otherId)) {
        threadMap.set(otherId, {
          contact: {
            id: String(otherId),
            name: otherName || 'User',
            email: otherEmail || '',
            role: otherRole || 'Member'
          },
          lastMessage: msg,
          unreadCount: 0,
          messages: []
        });
      }

      const thread = threadMap.get(otherId)!;
      thread.messages.push(msg);

      if (isIncoming && !msg.is_read) {
        thread.unreadCount++;
      }
    }

    this.chatThreads = Array.from(threadMap.values()).map(t => {
      t.messages.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
      t.lastMessage = t.messages[t.messages.length - 1];
      return t;
    });

    this.chatThreads.sort((a, b) => {
      const timeA = a.lastMessage ? new Date(a.lastMessage.created_at).getTime() : 0;
      const timeB = b.lastMessage ? new Date(b.lastMessage.created_at).getTime() : 0;
      return timeB - timeA;
    });

    if (this.selectedContact) {
      const targetId = String(this.selectedContact.id).toLowerCase();
      const activeThread = this.chatThreads.find(t => String(t.contact.id).toLowerCase() === targetId);
      if (activeThread) {
        this.activeConversationMessages = activeThread.messages;
      }
    } else if (this.chatThreads.length > 0) {
      this.selectContact(this.chatThreads[0].contact);
    }
  }

  selectContact(contact: any) {
    const norm = this.normalizeContact(contact);
    if (!norm) return;

    this.selectedContact = norm;
    const targetId = String(norm.id).toLowerCase();
    let thread = this.chatThreads.find(t => String(t.contact.id).toLowerCase() === targetId);

    if (thread) {
      this.activeConversationMessages = thread.messages;

      const currentUserId = String(this.currentUser?.id || this.currentUser?.ID || '').toLowerCase();
      for (const msg of thread.messages) {
        const recipientId = String(msg.recipient_id || '').toLowerCase();
        if (!msg.is_read && recipientId === currentUserId) {
          this.api.getMessageDetail(msg.id).subscribe({
            next: () => {
              msg.is_read = true;
              if (thread && thread.unreadCount > 0) thread.unreadCount--;
              this.refreshUnreadCount();
            }
          });
        }
      }
    } else {
      this.activeConversationMessages = [];
    }

    this.shouldScrollToBottom = true;
  }

  // ── Recipient Search Improvements ──────────────────────────────────────────

  onSearchInput() {
    this.searchError = null;
    const q = this.searchQuery.trim();
    this.messageFilterQuery = q;
    this.messageSearchSubject.next(q);

    if (!q) {
      this.searchResults = [];
      this.isSearching = false;
      return;
    }

    this.isSearching = true;
    this.recipientSearchSubject.next(q);
  }

  executeRecipientSearch(query: string) {
    const cached = this.searchCache.get(query.toLowerCase());
    if (cached) {
      const currentUserId = this.currentUser?.id || this.currentUser?.ID;
      this.searchResults = cached.filter(u => (u.id || u.ID) !== currentUserId);
      this.isSearching = false;
      this.checkSearchEmptyState(query);
      return;
    }

    this.api.searchUsers(query).subscribe({
      next: (results) => {
        const currentUserId = this.currentUser?.id || this.currentUser?.ID;
        const filtered = (results || []).filter(u => (u.id || u.ID) !== currentUserId);
        this.searchCache.set(query.toLowerCase(), filtered);
        this.searchResults = filtered;
        this.isSearching = false;
        this.checkSearchEmptyState(query);
      },
      error: (err) => {
        console.error('User search error:', err);
        this.searchResults = [];
        this.isSearching = false;
      }
    });
  }

  checkSearchEmptyState(query: string) {
    if (this.searchResults.length === 0) {
      if (query.length >= 3 && query.length <= 4 && !query.includes('@')) {
        this.searchError = 'No user found — try full email address.';
      } else if (query.includes('@')) {
        if (this.isValidEmail(query)) {
          this.searchExactEmail();
        } else {
          this.searchError = 'Please enter a valid email address.';
        }
      }
    }
  }

  isValidEmail(email: string): boolean {
    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return re.test(email);
  }

  searchExactEmail() {
    const email = this.searchQuery.trim();
    if (!email) return;

    if (!this.isValidEmail(email)) {
      this.searchError = 'Please enter a valid email address (e.g. user@domain.com).';
      return;
    }

    this.isSearching = true;
    this.searchError = null;

    this.api.getUserByEmail(email).subscribe({
      next: (user) => {
        this.isSearching = false;
        const currentId = this.currentUser?.id || this.currentUser?.ID;
        const userId = user?.id || user?.ID;
        if (user && userId === currentId) {
          this.searchError = 'You cannot chat with yourself.';
          return;
        }
        if (user) {
          this.startChatWithUser(user);
        }
      },
      error: (err) => {
        this.isSearching = false;
        this.searchError = err.error?.error || 'No registered user found with this exact email.';
      }
    });
  }

  startChatWithUser(user: any) {
    const norm = this.normalizeContact(user);
    if (!norm) {
      this.showToast('Unable to start chat with invalid contact.', 'error');
      return;
    }

    const currentUserId = this.currentUser?.id || this.currentUser?.ID;
    if (norm.id === currentUserId) {
      this.searchError = 'You cannot chat with yourself.';
      return;
    }

    let existingThread = this.chatThreads.find(t => t.contact.id === norm.id);
    if (!existingThread) {
      existingThread = {
        contact: norm,
        lastMessage: null,
        unreadCount: 0,
        messages: []
      };
      this.chatThreads.unshift(existingThread);
    }

    this.searchQuery = '';
    this.searchResults = [];
    this.searchError = null;
    this.activeTab = 'chats';
    this.activeSection = 'chats';
    this.selectContact(norm);
  }

  // ── Draft Auto-Save Logic ──────────────────────────────────────────────────

  onInputChange() {
    this.validationError = null;
    this.autoSaveDraftSubject.next();
  }

  performAutoSaveDraft() {
    if (!this.chatInput.trim() && !this.chatSubject.trim()) return;

    const draftData: any = {
      subject: this.chatSubject.trim(),
      body: this.chatInput.trim()
    };
    if (this.currentDraftId) {
      draftData.id = this.currentDraftId;
    }
    if (this.selectedContact?.id) {
      draftData.recipient_id = this.selectedContact.id;
    }

    this.api.saveMessageDraft(draftData).subscribe({
      next: (draft) => {
        if (draft && draft.id) {
          this.currentDraftId = draft.id;
          this.loadDrafts();
        }
      }
    });
  }

  continueEditingDraft(draft: any) {
    this.currentDraftId = draft.id;
    this.chatSubject = draft.subject || '';
    this.chatInput = draft.body || '';
    if (draft.subject) {
      this.showSubjectInput = true;
    }
    if (draft.recipient_id) {
      const contact = {
        id: draft.recipient_id,
        name: draft.recipient_name || 'User',
        email: draft.recipient_email || '',
        role: draft.recipient_role || 'Member'
      };
      this.startChatWithUser(contact);
    }
    this.activeSection = 'chats';
  }

  deleteDraftItem(draftId: string, event?: MouseEvent) {
    if (event) event.stopPropagation();
    this.api.deleteMessageDraft(draftId).subscribe({
      next: () => {
        if (this.currentDraftId === draftId) {
          this.currentDraftId = null;
          this.chatInput = '';
          this.chatSubject = '';
        }
        this.loadDrafts();
        this.showToast('Draft deleted.', 'success');
      }
    });
  }

  // ── Message Actions (Send, Read/Unread, Soft Delete, Restore) ─────────────

  sendMessage() {
    this.validationError = null;

    if (!this.selectedContact) {
      this.validationError = 'Please select a recipient to message.';
      this.showToast('Please select a recipient to message.', 'error');
      return;
    }

    const recipientId = this.selectedContact.id;
    const currentUserId = this.currentUser?.id || this.currentUser?.ID;

    if (!recipientId) {
      this.validationError = 'Invalid recipient selected.';
      this.showToast('Invalid recipient selected.', 'error');
      return;
    }

    if (recipientId === currentUserId) {
      this.validationError = 'You cannot send a message to yourself.';
      this.showToast('You cannot send a message to yourself.', 'error');
      return;
    }

    if (!this.chatInput.trim()) {
      this.validationError = 'Message body cannot be empty.';
      this.showToast('Please enter a message body before sending.', 'error');
      return;
    }

    const body = this.chatInput.trim();
    const subject = this.chatSubject.trim() || 'Chat Message';

    this.isSending = true;
    this.api.sendMessage(recipientId, subject, body, this.currentDraftId || undefined).subscribe({
      next: (res) => {
        this.isSending = false;
        this.chatInput = '';
        this.chatSubject = '';
        this.showSubjectInput = false;
        this.currentDraftId = null;
        this.validationError = null;

        let thread = this.chatThreads.find(t => t.contact.id === recipientId);
        if (!thread) {
          thread = {
            contact: this.selectedContact!,
            lastMessage: res,
            unreadCount: 0,
            messages: []
          };
          this.chatThreads.unshift(thread);
        }

        thread.messages.push(res);
        thread.lastMessage = res;
        this.activeConversationMessages = thread.messages;

        this.chatThreads.sort((a, b) => {
          const timeA = a.lastMessage ? new Date(a.lastMessage.created_at).getTime() : 0;
          const timeB = b.lastMessage ? new Date(b.lastMessage.created_at).getTime() : 0;
          return timeB - timeA;
        });

        this.loadDrafts();
        this.shouldScrollToBottom = true;
        this.showToast('Message sent successfully.', 'success');
      },
      error: (err) => {
        this.isSending = false;
        const errorMsg = err.error?.error || 'Message failed to send. Please try again.';
        this.validationError = errorMsg;
        this.showToast(errorMsg, 'error');
      }
    });
  }

  toggleRead(msg: any, event?: MouseEvent) {
    if (event) event.stopPropagation();
    const newStatus = !msg.is_read;
    this.api.toggleReadStatus(msg.id, newStatus).subscribe({
      next: () => {
        msg.is_read = newStatus;
        this.refreshUnreadCount();
        this.showToast(newStatus ? 'Marked as read.' : 'Marked as unread.', 'success');
      },
      error: () => {
        this.showToast('Failed to update read status.', 'error');
      }
    });
  }

  deleteMessage(msg: any, event?: MouseEvent) {
    if (event) event.stopPropagation();
    this.api.softDeleteMessage(msg.id).subscribe({
      next: () => {
        this.activeConversationMessages = this.activeConversationMessages.filter(m => m.id !== msg.id);
        this.loadData();
        this.showToast('Message moved to Trash.', 'success');
      },
      error: () => {
        this.showToast('Failed to delete message.', 'error');
      }
    });
  }

  restoreMessage(msg: any, event?: MouseEvent) {
    if (event) event.stopPropagation();
    this.api.restoreMessage(msg.id).subscribe({
      next: () => {
        this.loadData();
        this.showToast('Message restored from Trash.', 'success');
      },
      error: () => {
        this.showToast('Failed to restore message.', 'error');
      }
    });
  }

  toggleSubjectInput() {
    this.showSubjectInput = !this.showSubjectInput;
  }

  private scrollToBottom(): void {
    try {
      if (this.chatScrollContainer) {
        this.chatScrollContainer.nativeElement.scrollTop = this.chatScrollContainer.nativeElement.scrollHeight;
      }
    } catch (err) {
      console.error('Scroll to bottom error:', err);
    }
  }

  private showToast(msg: string, type: 'success' | 'error' = 'success') {
    this.toastMessage = msg;
    this.toastType = type;
    setTimeout(() => {
      this.toastMessage = null;
    }, 4000);
  }
}

