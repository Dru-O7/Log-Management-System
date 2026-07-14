const fs = require('fs');

// Fix login.component.html
let loginHtml = fs.readFileSync('src/app/components/login/login.component.html', 'utf8');
loginHtml = loginHtml.replace('</form>n>\n      </form>', '</form>');
fs.writeFileSync('src/app/components/login/login.component.html', loginHtml);

// Fix app.component.html
let appHtml = fs.readFileSync('src/app/app.component.html', 'utf8');

appHtml = appHtml.replace(/\[class\.border-\[var\(--primary\)\]\]="([^"]+)"\s+\[class\.bg-\[var\(--primary-light\)\]\]="([^"]+)"\s+\[class\.text-\[var\(--primary\)\]\]="([^"]+)"\s+\[class\.border-transparent\]="([^"]+)"\s+\[class\.text-\[var\(--text-secondary\)\]\]="([^"]+)"\s+\[class\.hover:bg-slate-50\]="([^"]+)"\s+\[class\.dark:hover:bg-slate-800\/40\]="([^"]+)"/g, 
  (match, p1) => {
    // p1 is the condition for active
    return `[ngClass]="${p1} ? 'border-[var(--primary)] bg-[var(--primary-light)] text-[var(--primary)]' : 'border-transparent text-[var(--text-secondary)] hover:bg-slate-50 dark:hover:bg-slate-800/40'"`;
  });

// Some might have md:w-20
appHtml = appHtml.replace(/\[class\.md:w-20\]="isSidebarCollapsed"/g, '[ngClass]="isSidebarCollapsed ? \'md:w-20\' : \'\'"');

fs.writeFileSync('src/app/app.component.html', appHtml);
console.log('Fixed files');
