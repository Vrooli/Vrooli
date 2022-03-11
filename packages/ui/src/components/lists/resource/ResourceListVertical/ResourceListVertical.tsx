// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { ResourceListItem } from 'components';
import { ResourceListVerticalProps } from '../types';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import { containerShadow } from 'styles';
import { Box } from '@mui/material';

export const ResourceListVertical = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    list,
    session,
}: ResourceListVerticalProps) => {

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selected, setSelected] = useState<any | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selected?.link}`, [selected]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, data: any) => {
        console.log('setting context anchor', ev.currentTarget, data);
        setContextAnchor(ev.currentTarget);
        setSelected(data);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelected(null);
    }, []);

    return (
        <Box sx={{
            ...containerShadow,
            borderRadius: '8px',
            overflow: 'overlay',
        }}>
            {list?.resources?.map((c: Resource, index) => (
                <ResourceListItem
                    key={`resource-card-${index}`}
                    data={c}
                    index={index}
                    session={session}
                />
            ))}
        </Box>
    )
}