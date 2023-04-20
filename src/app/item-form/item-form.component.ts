import { HttpClient } from '@angular/common/http';
import { Component } from '@angular/core';
import { FormBuilder, FormControl, FormGroup, Validators } from '@angular/forms';
import { StorageService } from '../services/storage.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-item-form',
  templateUrl: './item-form.component.html',
  styleUrls: ['./item-form.component.scss']
})
export class ItemFormComponent {
    form: FormGroup;
    
    constructor(public fb: FormBuilder,
      private http: HttpClient,
      private storage: StorageService,
      private router: Router) {

        this.form = this.fb.group({
          itemName: new FormControl('', [
            Validators.required,
          ]),
          itemDescription: new FormControl('', [
            Validators.required,
          ]),
          userID: new FormControl(''),
          image: new FormControl(null),
        });
    }

    fileName: string = '';
    fileUploaded: boolean = false;

    get f() {
      return this.form.controls;
    }

    onFileSelected(event: any): void {
      const file: File = event.target.files[0];

      if (file) {
        this.fileName = file.name;
        
        this.form.patchValue({
          image: file,
        });
        this.form.get('image')?.updateValueAndValidity();
      }
      this.fileUploaded = true;
    }

    addItem(): void {
      if (this.fileUploaded && this.f['itemName'].value !== '') {
        var id = this.storage.getUser().id_token;
        var formData: any = new FormData();
        formData.append('itemName', this.form.controls['itemName'].value);
        formData.append('itemDescription', this.form.controls['itemDescription'].value);
        formData.append('userID', id);
        formData.append('image', this.form.controls['image'].value);
        
        this.http.post("item/create", formData).subscribe({
          next: (res) => console.log(res),
          error: (err) => console.log(err),
        });

        this.router.navigate(['./user-profile']);
      }

      
    }

}