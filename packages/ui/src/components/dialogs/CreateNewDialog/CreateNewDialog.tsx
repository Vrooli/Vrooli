import {
    Dialog,
    DialogContent,
    DialogTitle,
    List,
    ListItem,
    ListItemText,
} from '@mui/material';
import { CreateNewDialogProps } from '../types';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';

enum ObjectType {
    Organization = 'Organization',
    Project = 'Project',
    Routine = 'Routine',
    Standard = 'Standard',
}

export const CreateNewDialog = ({
    handleClose,
    isOpen,
}: CreateNewDialogProps) => {
    const [, setLocation] = useLocation();

    const handleSelect = (type: ObjectType) => {
        switch (type) {
            case ObjectType.Organization:
                setLocation(`${APP_LINKS.SearchOrganizations}/add`);
                break;
            case ObjectType.Project:
                setLocation(`${APP_LINKS.SearchProjects}/add`);
                break;
            case ObjectType.Routine:
                setLocation(`${APP_LINKS.SearchRoutines}/add`);
                break;
            case ObjectType.Standard:
                setLocation(`${APP_LINKS.SearchStandards}/add`);
                break;
        }
    };

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            aria-labelledby="create-new-dialog-title"
            aria-describedby="create-new-dialog-description"
        >
            <DialogTitle 
                id="create-new-dialog-title" 
                sx={{
                    background: (t) => t.palette.primary.dark,
                    color: (t) => t.palette.primary.contrastText,
                }}
            >Create new....</DialogTitle>
            <DialogContent>
                <List sx={{ marginTop: 'auto' }}>
                    {Object.values(ObjectType).map((type) => (
                        <ListItem
                            key={type}
                            button
                            onClick={() => handleSelect(type)}
                        >
                            <ListItemText primary={type} />
                        </ListItem>
                    ))}
                </List>
            </DialogContent>
        </Dialog>
    )
}