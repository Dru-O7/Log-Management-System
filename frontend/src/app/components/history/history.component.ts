import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './history.component.html',
  styleUrls: ['./history.component.css']
})
export class HistoryComponent implements OnInit {
  currentUser: any = null;
  documents: any[] = [];
  filteredDocuments: any[] = [];
  documentTypes: any[] = [];
  
  // Filter bindings
  searchText: string = '';
  selectedCategory: string = 'All';
  selectedPriority: string = 'All';
  selectedStatus: string = 'All';
  loadingList: boolean = false;

  // Selected document details & history
  selectedDoc: any = null;
  selectedDocHistory: any[] = [];
  loadingDetails: boolean = false;
  detailsError: string = '';

  constructor(
    private api: ApiService,
    private auth: AuthService,
    public router: Router
  ) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }
    this.loadDocumentTypes();
    this.loadAllDocuments();
  }

  loadDocumentTypes() {
    this.api.getDocumentTypes().subscribe({
      next: (types) => {
        this.documentTypes = types;
      },
      error: (err) => console.error('Failed to load document types:', err)
    });
  }

  loadAllDocuments() {
    this.loadingList = true;
    this.api.getDocuments(this.currentUser.ID).subscribe({
      next: (docs) => {
        this.documents = docs || [];
        this.applyFilter();
        this.loadingList = false;
      },
      error: (err) => {
        console.error('Failed to load documents:', err);
        this.loadingList = false;
      }
    });
  }

  applyFilter() {
    this.filteredDocuments = this.documents.filter(doc => {
      const matchSearch = !this.searchText || 
        doc.Title.toLowerCase().includes(this.searchText.toLowerCase()) ||
        doc.UniqueNumber.toLowerCase().includes(this.searchText.toLowerCase()) ||
        (doc.Description && doc.Description.toLowerCase().includes(this.searchText.toLowerCase()));
      const matchCategory = this.selectedCategory === 'All' || doc.Category === this.selectedCategory;
      const matchPriority = this.selectedPriority === 'All' || doc.Priority === this.selectedPriority;
      const matchStatus = this.selectedStatus === 'All' || doc.Status === this.selectedStatus;
      return matchSearch && matchCategory && matchPriority && matchStatus;
    });

    // If currently selected document is no longer in filtered list, deselect
    if (this.selectedDoc && !this.filteredDocuments.some(d => d.ID === this.selectedDoc.ID)) {
      this.selectedDoc = null;
      this.selectedDocHistory = [];
    }
  }

  selectDocument(doc: any) {
    this.router.navigate(['/details', doc.ID]);
  }

  getHoldingDuration(assignedAtStr: string): string {
    if (!assignedAtStr) return '0m';
    const assigned = new Date(assignedAtStr);
    const now = new Date();
    const diffMs = now.getTime() - assigned.getTime();
    const diffMins = Math.max(0, Math.floor(diffMs / 60000));
    
    if (diffMins < 60) return `${diffMins}m`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h`;
    return `${Math.floor(diffHours / 24)}d`;
  }
}
