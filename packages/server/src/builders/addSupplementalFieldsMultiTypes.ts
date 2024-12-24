import { SessionUser } from "@local/shared";
import { addSupplementalFields } from "./addSupplementalFields";
import { ApiEndpointInfo } from "./types";

type DataShape = { [key: string]: any[] };

/**
 * Combines addSupplementalFields calls for multiple object types
 * @param data Object of arrays, where each value is a list of the same object type queried from the database
 * @param partial API endpoint info object with the same keys as data, and values equal to the partial info for that object type
 * @param userData Requesting user's data
 * @returns Object in same shape as data, but with each value containing supplemental data
 */
export async function addSupplementalFieldsMultiTypes<
    TData extends DataShape,
    TPartial extends { [K in keyof TData]: ApiEndpointInfo }
>(
    data: TData,
    partial: TPartial,
    userData: SessionUser | null,
): Promise<{ [K in keyof TData]: any[] }> {
    // Flatten data object into an array and create an array of partials that match the data array
    const combinedData: any[] = [];
    const combinedPartialInfo: ApiEndpointInfo[] = [];
    for (const key in data) {
        for (const item of data[key]) {
            combinedData.push(item);
            combinedPartialInfo.push(partial[key]);
        }
    }

    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(userData, combinedData, combinedPartialInfo);

    // Convert combinedResult into object in the same shape as data, but with each value containing supplemental data
    const formatted = {} as { [K in keyof TData]: any[] };
    let start = 0;
    for (const key in data) {
        const end = start + data[key].length;
        formatted[key] = combinedResult.slice(start, end);
        start = end;
    }
    return formatted;
};
