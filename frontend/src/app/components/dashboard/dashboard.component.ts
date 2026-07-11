import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  documents: any[] = [];
  filteredDocuments: any[] = [];
  currentUser: any = null;
  searchText: string = '';
  documentTypes: any[] = [];
  selectedFolder: string = 'All';

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }
    this.loadDocumentTypes();
    this.loadDocuments();
  }

  loadDocumentTypes() {
    this.api.getDocumentTypes().subscribe({
      next: (types) => {
        this.documentTypes = types;
      }
    });
  }

  loadDocuments() {
    this.api.getDocuments(this.currentUser.ID, this.searchText).subscribe({
      next: (docs) => {
        this.documents = docs;
        this.applyFilter();
      }
    });
  }

  onSearchChange() {
    this.loadDocuments();
  }

  selectFolder(folderName: string) {
    this.selectedFolder = folderName;
    this.applyFilter();
  }

  applyFilter() {
    if (this.selectedFolder === 'All') {
      this.filteredDocuments = this.documents;
    } else {
      this.filteredDocuments = this.documents.filter(doc => 
        doc.Category?.toLowerCase() === this.selectedFolder.toLowerCase()
      );
    }
  }

  getFolderCount(folderName: string): number {
    if (folderName === 'All') {
      return this.documents.length;
    }
    return this.documents.filter(doc => 
      doc.Category?.toLowerCase() === folderName.toLowerCase()
    ).length;
  }

  goToUpload() {
    this.router.navigate(['/upload']);
  }

  goToDetails(id: string) {
    this.router.navigate(['/details', id]);
  }
}
